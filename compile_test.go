package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

var inputCompile events.APIGatewayProxyRequest = events.APIGatewayProxyRequest{
	Path: "/compile",
	Body: `{"version":2,"body":"package main \n \nimport ( \n    \"fmt\" \n) \n    \nfunc main() { \n    fmt.Println(\"Hello, playground\") \n}"}`,
}

var expectedCompile string = `{"compiled":";(function() {\n  var pkg = {};\n  pkg[\"playground\"] = (function() {\n    function main () {\n      console.log.apply(console.log, [\"Hello, playground\"])\n    };\n    return {\n      main: main\n    };\n  })();\n  return pkg[\"playground\"].main();\n})()","error":""}`

func TestCompile(t *testing.T) {
	out, err := Handle(inputCompile)
	if err != nil {
		t.Error(err)
	}
	if out.StatusCode != 200 {
		t.Errorf("error in handler: " + out.Body)
	}

	if out.Body != expectedCompile {
		t.Errorf("invalid expected body data")
	}
}
