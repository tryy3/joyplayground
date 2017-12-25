package main

import (
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"testing"
)

var inputShare string = `{
	"path": "/share",
	"body": "package main \n \nimport ( \n    \"fmt\" \n) \n    \nfunc main() { \n    fmt.Println(\"Hello, playground test\") \n}"
}`

var shareID string = "IAAEPbTy59"

func TestShare(t *testing.T) {
	ctx := &runtime.Context{}
	out, err := Handle([]byte(inputShare), ctx)
	if err != nil {
		t.Error(err)
	}
	if out.(Response).StatusCode != 200 {
		t.Errorf("error in handler: " + out.(Response).Body)
	}
	if out.(Response).Body != shareID {
		t.Errorf("Expected %s got %s when testing share", shareID, out.(Response).Body)
	}
}

var inputP string = `{
	"path": "/p/IAAEPbTy59"	
}`

var expectedP string = "package main \n \nimport ( \n    \"fmt\" \n) \n    \nfunc main() { \n    fmt.Println(\"Hello, playground test\") \n}"

func TestP(t *testing.T) {
	ctx := &runtime.Context{}
	out, err := Handle([]byte(inputP), ctx)
	if err != nil {
		t.Error(err)
	}
	if out.(Response).StatusCode != 200 {
		t.Errorf("error in handler: " + out.(Response).Body)
	}
	if out.(Response).Body != expectedP {
		t.Errorf("Expected %s got %s when testing share", shareID, out.(Response).Body)
	}
}
