package main

import (
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"testing"
)

var inputFmt string = `{
	"path": "/fmt",
	"body": "{\"body\":\"package main \\n \\nimport ( \\n    \\\"fmt\\\" \\n) \\n    \\nfunc main() { \\nfmt.Println(\\\"Hello, playground\\\") \\n}\"}"
}`

var expectedFmtBody string = "{\"Body\":\"package main\\n\\nimport (\\n\\t\\\"fmt\\\"\\n)\\n\\nfunc main() {\\n\\tfmt.Println(\\\"Hello, playground\\\")\\n}\\n\",\"Error\":\"\"}"

func TestFmt(t *testing.T) {
	ctx := &runtime.Context{}
	out, err := Handle([]byte(inputFmt), ctx)
	if err != nil {
		t.Error(err)
	}
	if out.(Response).StatusCode != 200 {
		t.Errorf("error in handler: " + out.(Response).Body)
	}
	if out.(Response).Body != expectedFmtBody {
		t.Errorf("Invalid expected body data")
	}
}

var inputFmtImport string = `{
	"path": "/fmt",
	"body": "{\"body\":\"package main\\n\\nfunc main() {\\n\\tfmt.Println(\\\"Hello, playground\\\")\\n}\\n\",\"imports\":true}"
}`

var expectedFmtImportBody string = "{\"Body\":\"package main\\n\\nimport \\\"fmt\\\"\\n\\nfunc main() {\\n\\tfmt.Println(\\\"Hello, playground\\\")\\n}\\n\",\"Error\":\"\"}"

func TestFmtImport(t *testing.T) {
	ctx := &runtime.Context{}
	out, err := Handle([]byte(inputFmtImport), ctx)
	if err != nil {
		t.Error(err)
	}
	if out.(Response).StatusCode != 200 {
		t.Errorf("error in handler: " + out.(Response).Body)
	}
	if out.(Response).Body != expectedFmtImportBody {
		t.Errorf("Invalid expected body data")
	}
}
