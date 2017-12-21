package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
)

type Response struct {
	StatusCode int               `json:"statusCode"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
}

type Request struct {
	Body string `json:"body"`
	Path string `json:"path"`
}

type BodyRequest struct {
	Body string
}

type ResponseEvents struct {
	Errors string
	Events []Event
}

type fmtRequest struct {
	Body    string
	Imports bool
}

type fmtResponse struct {
	Body  string
	Error string
}

func successResponse(msg string) Response {
	return Response{
		StatusCode: 200,
		Body:       msg,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}
}

func Handle(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	p := os.Getenv("PATH")
	dir, err := os.Getwd()
	if err != nil {
		log.Printf("error getting working directory: %v\n", err)
		return nil, fmt.Errorf("error getting working directory: %v", err)
	}
	if err := os.Setenv("PATH", p+":"+dir); err != nil {
		log.Printf("error setting PATH: %v\n", err)
		return nil, fmt.Errorf("error setting PATH: %v", err)
	}

	var req Request
	if err := json.Unmarshal(evt, &req); err != nil {
		log.Printf("error decoding request: %v\n", err)
		return nil, fmt.Errorf("error decoding request: %v", err)
	}

	switch req.Path {
	case "/compile":
		return compileHandler(req)
	case "/fmt":
		return fmtHandler(req)
	case "/share":
		return shareHandler(req)
	default:
		if strings.Contains(req.Path, "/p") {
			return pHandler(req)
		}
		return successResponse("Hello Worldsss"), nil
	}

}
