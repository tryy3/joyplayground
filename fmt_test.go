package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

var inputFmt events.APIGatewayProxyRequest = events.APIGatewayProxyRequest{
	Path: "/fmt",
	Body: `"{\"body\":\"package main \\n \\nimport ( \\n    \\\"fmt\\\" \\n) \\n    \\nfunc main() { \\nfmt.Println(\\\"Hello, playground\\\") \\n}\"}"`,
}

var expectedFmtBody string = "{\"Body\":\"package main\\n\\nimport (\\n\\t\\\"fmt\\\"\\n)\\n\\nfunc main() {\\n\\tfmt.Println(\\\"Hello, playground\\\")\\n}\\n\",\"Error\":\"\"}"

func TestFmt(t *testing.T) {
	out, err := Handle(inputFmt)
	if err != nil {
		t.Error(err)
	}
	if out.StatusCode != 200 {
		t.Errorf("error in handler: " + out.Body)
	}
	if out.Body != expectedFmtBody {
		t.Errorf("Invalid expected body data")
	}
}

var inputFmtImport events.APIGatewayProxyRequest = events.APIGatewayProxyRequest{
	Path: "/fmt",
	Body: `"{\"body\":\"package main\\n\\nfunc main() {\\n\\tfmt.Println(\\\"Hello, playground\\\")\\n}\\n\",\"imports\":true}"`,
}

var expectedFmtImportBody string = "{\"Body\":\"package main\\n\\nimport \\\"fmt\\\"\\n\\nfunc main() {\\n\\tfmt.Println(\\\"Hello, playground\\\")\\n}\\n\",\"Error\":\"\"}"

func TestFmtImport(t *testing.T) {
	out, err := Handle(inputFmtImport)
	if err != nil {
		t.Error(err)
	}
	if out.StatusCode != 200 {
		t.Errorf("error in handler: " + out.Body)
	}
	if out.Body != expectedFmtImportBody {
		t.Errorf("Invalid expected body data")
	}
}
