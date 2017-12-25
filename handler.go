package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
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

// ResponseEvents is the expected data when outputting the response from compiling golang code
type ResponseEvents struct {
	Errors string
	Events []Event
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
func successResponse(msg string) Response {
	return Response{
		StatusCode: 200,
		Body:       msg,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}
}

// errorResponse is a wrapper for outputting an error
func errorResponse(msg string, statusCode int) Response {
	return Response{
		StatusCode: statusCode,
		Body:       msg,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}
}

// Handle is the main function, it will check the Path in evt json.RawMessage and then determine what function to run
func Handle(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
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

	// Unmarshal the incoming request
	var req Request
	if err := json.Unmarshal(evt, &req); err != nil {
		log.Printf("error decoding request: %v\n", err)
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("syntax error at byte offset %d", e.Offset)
			return errorResponse(fmt.Sprintf("error decoding request: syntax error at byte offset %d", e.Offset), 400), nil
		}
		return errorResponse(fmt.Sprintf("error decoding request: %v", err), 400), nil
	}

	// Determine what function to run based on path
	if strings.Contains(req.Path, "/compile") {
		return compileHandler(req)
	} else if strings.Contains(req.Path, "/fmt") {
		return fmtHandler(req)
	} else if strings.Contains(req.Path, "/share") {
		return shareHandler(req)
	} else if strings.Contains(req.Path, "/p") {
		return pHandler(req)
	}

	return errorResponse("Invalid path", 404), nil
}
