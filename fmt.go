package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/tools/imports"
)

// fmtHandler handles formatting golang code
func fmtHandler(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var (
		out []byte
		err error
		req fmtRequest
	)

	// unmarshal the incoming json request
	if err := json.Unmarshal([]byte(r.Body), &req); err != nil {
		log.Printf("error decoding request: %v\n", err)
		return errorResponse(fmt.Sprintf("error decoding request: %v", err), 400), nil
	}

	// check if we are gonna add imports to or not
	if req.Imports {
		out, err = imports.Process("prog.go", []byte(req.Body), nil)
	} else {
		out, err = format.Source([]byte(req.Body))
	}

	// check if there were any errors
	var resp fmtResponse
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Body = string(out)
	}

	// marshal the formatted golang code and output it
	output, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error encoding response: %v\n", err)
		return errorResponse(fmt.Sprintf("error encoding response: %v", err), 400), nil
	}
	return successResponse(string(output)), nil
}
