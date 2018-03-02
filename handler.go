package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is the expected data when outputting to a HTTP request
type Response struct {
	StatusCode int               `json:"statusCode"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
}

// Request is the expected data from an incoming event
type Request struct {
	Body string `json:"body"`
	Path string `json:"path"`
}

// BodyRequest is the expected data for the Body in Request struct when compiling golang code
type BodyRequest struct {
	Body string
}

// fmtRequest is the expected data for the Body in Request struct when formatting golang code
type fmtRequest struct {
	Body    string
	Imports bool
}

// fmtResponse is the expected data when outputting the response from formatting golang code
type fmtResponse struct {
	Body  string
	Error string
}

// successResponse is a wrapper for outputting a message
func successResponse(msg string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       msg,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}
}

// errorResponse is a wrapper for outputting an error
func errorResponse(msg string, statusCode int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
		Body: msg,
	}
}

// Handle is the main function, it will check the Path in evt json.RawMessage and then determine what function to run
func Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Add working directory to PATH, used for lambda functions.
	p := os.Getenv("PATH")
	dir, err := os.Getwd()
	if err != nil {
		log.Printf("error getting working directory: %v\n", err)
		return errorResponse(fmt.Sprintf("error getting working directory: %v", err), 400), nil
	}
	if err := os.Setenv("PATH", p+":"+dir); err != nil {
		log.Printf("error setting PATH: %v\n", err)
		return errorResponse(fmt.Sprintf("error setting PATH: %v", err), 400), nil
	}

	// Determine what function to run based on path
	if strings.Contains(request.Path, "/compile") {
		return compileHandler(request)
	} else if strings.Contains(request.Path, "/fmt") {
		return fmtHandler(request)
	} else if strings.Contains(request.Path, "/share") {
		return shareHandler(request)
	} else if strings.Contains(request.Path, "/p") {
		return pHandler(request)
	} else if strings.Contains(request.Path, "/js") {
		return jsHandler(request)
	}

	return errorResponse("Invalid path", 404), nil
}

func main() {
	lambda.Start(Handle)
}
