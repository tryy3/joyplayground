package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// shareHandler handles all of the requests when sharing golang code
func shareHandler(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, err := storeSnippet("", []byte(r.Body))
	if err != nil {
		log.Printf("couldn't store snippet: %v", err)
		return errorResponse("Internal error", 500), nil
	}

	return successResponse(id), nil
}

// pHandler handles all of the requests when someone tries to retrieve golang code from an ID
func pHandler(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get the ID from path
	id := r.Path[strings.Index(r.Path, "p/")+2:]
	err := validateID(id)
	if err != nil {
		log.Println("Unexpected id format.")
		return errorResponse("Unexpected if format.", 400), nil
	}

	// Support for multiple snippet systems
	if snippet, err := getSnippetFromS3Store("", id); err == nil { // Check if we have the snippet on s3 first.
		return successResponse(snippet), nil
	} else if snippet, err = getSnippetFromLocalStore(id); err == nil { // Check if we have the snippet locally.
		return successResponse(snippet), nil
	} else if snippet, err = getSnippetFromGoPlayground(id); err == nil { // If not found locally, try the Go Playground.
		return successResponse(snippet), nil
	}
	log.Printf("error retrieving the snippet: %v", err)
	return errorResponse(fmt.Sprintf("error retrieving the snippet: %v", err), 400), nil
}

// storeSnippet stores snippet in s3.
// It returns the id assigned to the snippet.
func storeSnippet(key string, body []byte) (id string, err error) {
	id = snippetBodyToID(body)

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	// Create an uploader with S3 client and default options
	uploader := s3manager.NewUploaderWithClient(svc)

	// Upload input parameters
	upParams := &s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(key + id),
		Body:   bytes.NewReader(body),
	}

	// Perform an upload.
	_, err = uploader.Upload(upParams)
	if err != nil {
		return "", fmt.Errorf("error sending s3 object: %v", err)
	}
	return id, nil
}

// snippetBodyToID mimics the mapping scheme used by the Go Playground.
func snippetBodyToID(body []byte) string {
	// This is the actual salt value used by Go Playground, it comes from
	// https://code.google.com/p/go-playground/source/browse/goplay/share.go#18.
	// See https://github.com/gopherjs/snippet-store/pull/1#discussion_r22512198 for more details.
	const salt = "[replace this with something unique]"
	h := sha1.New()
	io.WriteString(h, salt)
	h.Write(body)
	sum := h.Sum(nil)
	return base64.URLEncoding.EncodeToString(sum)[:10]
}

// validateID returns an error if id is of unexpected format.
func validateID(id string) error {
	if len(id) != 10 {
		return fmt.Errorf("id length is %v instead of 10", len(id))
	}

	for _, b := range []byte(id) {
		ok := ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') || b == '-' || b == '_'
		if !ok {
			return fmt.Errorf("id contains unexpected character %+q", b)
		}
	}

	return nil
}

// getSnippetFromS3Store tries to get the snippet with given id from s3 bucket.
func getSnippetFromS3Store(key, id string) (string, error) {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(key + id),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		return "", fmt.Errorf("error retrieving s3 object: %v", err)
	}

	snippet, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return "", fmt.Errorf("error reading s3 object: %v", err)
	}
	return string(snippet), err
}

// getSnippetFromLocalStore tries to get the snippet with given id from local store.
func getSnippetFromLocalStore(id string) (string, error) {
	tmpDir := os.Getenv("LOCAL_STORE")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}

	data, err := ioutil.ReadFile(path.Join(tmpDir, id))
	return string(data), err
}

const userAgent = "gopherjs.org/play/ playground snippet fetcher"

// getSnippetFromGoPlayground tries to get the snippet with given id from the Go Playground.
func getSnippetFromGoPlayground(id string) (string, error) {
	req, err := http.NewRequest("GET", "https://play.golang.org/p/"+id+".go", nil)
	if err != nil {
		return "", fmt.Errorf("error creating new http request: %v", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending http request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", fmt.Errorf("go playground returned unexpected status code %v", resp.StatusCode)
	}

	snippet, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading data from http request: %v", err)
	}
	return string(snippet), err
}
