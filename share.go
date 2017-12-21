package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func shareHandler(r Request) (interface{}, error) {
	id, err := storeSnippet([]byte(r.Body))
	if err != nil {
		log.Printf("couldn't store snippet: %v", err)
		return nil, errors.New("Server error.")
	}

	return successResponse(id), nil
}

func pHandler(r Request) (interface{}, error) {
	id := r.Path[len("/p/"):]
	err := validateID(id)
	if err != nil {
		log.Println("Unexpected id format.")
		return nil, errors.New("Unexpected id format.")
	}

	if snippet, err := getSnippetFromS3Store(id); err == nil { // Check if we have the snippet locally first.
		return successResponse(snippet), nil
	} else if snippet, err = getSnippetFromGoPlayground(id); err == nil { // If not found locally, try the Go Playground.
		return successResponse(snippet), nil
	}
	log.Printf("error retrieving the snippet: %v", err)
	return nil, err
}

// storeSnippet stores snippet in local storage.
// It returns the id assigned to the snippet.
func storeSnippet(body []byte) (id string, err error) {
	id = snippetBodyToID(body)

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(bytes.NewReader(body)),
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(id),
	}

	_, err = svc.PutObject(input)
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
func getSnippetFromS3Store(id string) (string, error) {
	svc := s3.New(session.New())
	input := &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(id),
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
		return "", fmt.Errorf("Go Playground returned unexpected status code %v", resp.StatusCode)
	}

	snippet, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading data from http request: %v", err)
	}
	return string(snippet), err
}
