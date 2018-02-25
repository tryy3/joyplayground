package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

var inputShare events.APIGatewayProxyRequest = events.APIGatewayProxyRequest{
	Path: "/share",
	Body: `"package main \n \nimport ( \n    \"fmt\" \n) \n    \nfunc main() { \n    fmt.Println(\"Hello, playground test\") \n}"`,
}

var shareID string = "Tf4IB75zMW"

func TestShare(t *testing.T) {
	out, err := Handle(inputShare)
	if err != nil {
		t.Error(err)
	}
	if out.StatusCode != 200 {
		t.Errorf("error in handler: " + out.Body)
	}
	if out.Body != shareID {
		t.Errorf("Expected %s got %s when testing share", shareID, out.Body)
	}
}

var inputP events.APIGatewayProxyRequest = events.APIGatewayProxyRequest{
	Path: "/p/IAAEPbTy59",
}

var expectedP string = "package main \n \nimport ( \n    \"fmt\" \n) \n    \nfunc main() { \n    fmt.Println(\"Hello, playground test\") \n}"

func TestP(t *testing.T) {
	out, err := Handle(inputP)
	if err != nil {
		t.Error(err)
	}
	if out.StatusCode != 200 {
		t.Errorf("error in handler: " + out.Body)
	}
	if out.Body != expectedP {
		t.Errorf("Expected %s got %s when testing share", shareID, out.Body)
	}
}
