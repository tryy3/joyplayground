package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/aws/aws-lambda-go/events"
)

// jsHandler handles all of the request of serving compiled code
func jsHandler(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get the ID from path
	id := r.Path[strings.Index(r.Path, "js/")+3:]
	err := validateID(id)
	if err != nil {
		log.Println("Unexpected id format: " + r.Path)
		return errorResponse("Unexpected if format.", 400), nil
	}

	// Support for multiple snippet systems
	if snippet, err := getSnippetFromS3Store("js/", id); err == nil { // Check if the compiled code exists on s3
		return jsOutput(snippet)
	} else if snippet, err = getSnippetFromS3Store("", id); err == nil { // Check if we have the snippet on s3 first.
		compiled, err := compileAndRun(snippet)
		if err != nil {
			return errorResponse(fmt.Sprintf("error trying to compile code: %v", err), 400), nil
		} else if compiled.Error != "" {
			return errorResponse(fmt.Sprintf("error trying to compile code: %s", compiled.Error), 400), nil
		}
		return jsOutput(snippet)
	} else if snippet, err = getSnippetFromLocalStore(id); err == nil { // Check if we have the snippet locally.
		compiled, err := compileAndRun(snippet)
		if err != nil {
			return errorResponse(fmt.Sprintf("error trying to compile code: %v", err), 400), nil
		} else if compiled.Error != "" {
			return errorResponse(fmt.Sprintf("error trying to compile code: %s", compiled.Error), 400), nil
		}
		return jsOutput(snippet)
	} else if snippet, err = getSnippetFromGoPlayground(id); err == nil { // If not found locally, try the Go Playground.
		compiled, err := compileAndRun(snippet)
		if err != nil {
			return errorResponse(fmt.Sprintf("error trying to compile code: %v", err), 400), nil
		} else if compiled.Error != "" {
			return errorResponse(fmt.Sprintf("error trying to compile code: %s", compiled.Error), 400), nil
		}
		return jsOutput(snippet)
	}
	log.Printf("error retrieving the snippet: %v", err)
	return errorResponse(fmt.Sprintf("error retrieving the snippet: %v", err), 400), nil
}

var jsTemplate = `<html>
	<head></head>
	<body>
		<script>
			if (typeof window.joyOutput === 'undefined') window.joyOutput = {}
			window.joyOutput.log = function() {
				console.log.apply(console.log, arguments)
				window.top.postMessage(Array.from(arguments).join(" ") + "\n", "*")
			}
			{{.Snippet}}
		</script>
	</body>
</html>`

type jsTemplateStruct struct {
	Snippet string
}

// jsOutput is a simple wrapper for outputting HTML text to the request
func jsOutput(snippet string) (events.APIGatewayProxyResponse, error) {
	tmpl, err := template.New("snippet").Parse(jsTemplate)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	var writer bytes.Buffer
	err = tmpl.Execute(&writer, jsTemplateStruct{Snippet: strings.Replace(snippet, "console.log", "window.joyOutput.log", -1)})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"content-type": "text/html; charset=UTF-8"},
		Body:       writer.String(),
	}, nil
}
