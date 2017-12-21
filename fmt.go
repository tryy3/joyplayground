package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"golang.org/x/tools/imports"
	"log"
)

func fmtHandler(r Request) (interface{}, error) {
	var (
		out []byte
		err error
		req fmtRequest
	)
	if err := json.Unmarshal([]byte(r.Body), &req); err != nil {
		log.Printf("error decoding request: %v\n", err)
		return nil, fmt.Errorf("error decoding request: %v", err)
	}
	if req.Imports {
		out, err = imports.Process("prog.go", []byte(req.Body), nil)
	} else {
		out, err = format.Source([]byte(req.Body))
	}

	var resp fmtResponse
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Body = string(out)
	}

	output, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error encoding response: %v\n", err)
		return nil, fmt.Errorf("error encoding response: %v", err)
	}
	return successResponse(string(output)), nil
}
