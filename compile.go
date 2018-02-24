package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/aws/aws-lambda-go/events"
	apibuild "github.com/matthewmueller/joy/api/build"
)

// CompileOutput is the expected data when outputting compiled code
type CompileOutput struct {
	Compiled string `json:"compiled"`
	Error    string `json:"error"`
}

// compileHandler handles all of the incoming requests for compile
func compileHandler(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// decode the request body
	var req BodyRequest
	if err := json.Unmarshal([]byte(r.Body), &req); err != nil {
		log.Printf("error decoding body: %v\n", err)
		return errorResponse(fmt.Sprintf("error decoding body: %v", err), 400), nil
	}

	// compile and run the code
	resp, err := compileAndRun(&req)
	if err != nil {
		log.Printf("Compile error: %v\n", err)
		return events.APIGatewayProxyResponse{}, err
	}

	// encode the output of the compile
	out, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error encoding response: %v\n", err)
		return errorResponse(fmt.Sprintf("error encoding response: %v", err), 400), nil
	}
	return successResponse(string(out)), nil
}

// compileAndRun will compile the golang code by first creating a temp file
// and then using joy to compile the code to javascript
func compileAndRun(req *BodyRequest) (*CompileOutput, error) {
	// create a new directory in the OS's temp dir
	tmpDir, err := ioutil.TempDir("", "sandbox")
	if err != nil {
		return nil, fmt.Errorf("error creating temp directory: %v", err)
	}

	// remove the tmpDir folder when function ends
	defer os.RemoveAll(tmpDir)

	// create a new main.go file in the tmpDir
	in := filepath.Join(tmpDir, "main.go")
	if err := ioutil.WriteFile(in, []byte(req.Body), 0400); err != nil {
		return nil, fmt.Errorf("error creating temp file %q: %v", in, err)
	}

	// start parsing the main.go with joy
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, in, nil, parser.PackageClauseOnly)
	if err == nil && f.Name.Name != "main" {
		return &CompileOutput{Error: "package name must be main"}, nil
	}

	ctx := trap(syscall.SIGINT, syscall.SIGTERM)

	/// compile the code with joy
	files, err := apibuild.Build(&apibuild.Config{
		Context:  ctx,
		Packages: []string{in},
		JoyPath:  "/tmp",
	})
	if err != nil {
		return &CompileOutput{Error: fmt.Sprintf("error building code: %v", err)}, nil
	} else if len(files) != 1 {
		return &CompileOutput{Error: fmt.Sprintf("joy run expects only 1 main file, but received %d files", len(files))}, nil
	}

	regPkgName, err := regexp.Compile(`pkg\[(.+)]`)
	if err != nil {
		return &CompileOutput{Error: fmt.Sprintf("error compiling regex: %s", err)}, nil
	}

	// rewrite any mentions of the tmpdir and replace with pkg["playground"]
	// TODO: Maybe switch out "playground" with something else?
	output := &CompileOutput{
		Compiled: regPkgName.ReplaceAllString(files[0].Source(), "pkg[\"playground\"]"),
	}

	return output, nil
}

func trap(sig ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, sig...)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case <-c:
			cancel()
		}
	}()

	return ctx
}
