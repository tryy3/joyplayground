package main

import (
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"testing"
)

var inputCompile string = `{
	"path": "/compile",
	"body": "{\"version\":2,\"body\":\"package main \\n \\nimport ( \\n    \\\"fmt\\\" \\n) \\n    \\nfunc main() { \\n    fmt.Println(\\\"Hello, playground\\\") \\n}\"}"
}`

func TestCompile(t *testing.T) {
	ctx := &runtime.Context{}
	out, err := Handle([]byte(inputCompile), ctx)
	if err != nil {
		t.Error(err)
	}
	if out.(Response).StatusCode != 200 {
		t.Errorf("error in handler: " + out.(Response).Body)
	}
}
