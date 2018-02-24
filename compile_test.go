package main

import (
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

var inputCompile events.APIGatewayProxyRequest = events.APIGatewayProxyRequest{
	Path: "/compile",
	Body: `{"version":2,"body":"package main \n \nimport ( \n    \"fmt\" \n) \n    \nfunc main() { \n    fmt.Println(\"Hello, playground\") \n}"}`,
}

func TestCompile(t *testing.T) {
	out, err := Handle(inputCompile)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(out.Body)
	if out.StatusCode != 200 {
		t.Errorf("error in handler: " + out.Body)
	}
}
