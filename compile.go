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
	"time"

	apibuild "github.com/matthewmueller/joy/api/build"
)

type Event struct {
	Message string
	Kind    string        // "stdout" or "stderr"
	Delay   time.Duration // time to wait before printing Message
}

type event struct {
	msg  []byte
	kind string
	time time.Time
}

func compileHandler(r Request) (interface{}, error) {
	var req BodyRequest
	if err := json.Unmarshal([]byte(r.Body), &req); err != nil {
		log.Printf("error decoding request: %v\n", err)
		return nil, fmt.Errorf("error decoding request: %v", err)
	}

	resp, err := compileAndRun(&req)
	if err != nil {
		log.Printf("Compile error: %v\n", err)
		return nil, err
	}

	out, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error encoding response: %v\n", err)
		return nil, fmt.Errorf("error encoding response: %v", err)
	}
	return Response{StatusCode: 200, Body: string(out)}, nil
}

func compileAndRun(req *BodyRequest) (*ResponseEvents, error) {
	tmpDir, err := ioutil.TempDir("", "sandbox")
	if err != nil {
		return nil, fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	in := filepath.Join(tmpDir, "main.go")
	if err := ioutil.WriteFile(in, []byte(req.Body), 0400); err != nil {
		return nil, fmt.Errorf("error creating temp file %q: %v", in, err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, in, nil, parser.PackageClauseOnly)
	if err == nil && f.Name.Name != "main" {
		return &ResponseEvents{Errors: "package name must be main"}, nil
	}

	ctx := trap(syscall.SIGINT, syscall.SIGTERM)

	files, err := apibuild.Build(&apibuild.Config{
		Context:  ctx,
		Packages: []string{in},
		JoyPath:  "/tmp",
	})

	if err != nil {
		return &ResponseEvents{Errors: fmt.Sprintf("error building code: %v", err)}, nil
	} else if len(files) != 1 {
		return &ResponseEvents{Errors: fmt.Sprintf("joy run expects only 1 main file, but received %d files", len(files))}, nil
	}

	var events []Event

	regPkgName, err := regexp.Compile(`pkg\[(.+)\]`)
	if err != nil {
		return &ResponseEvents{Errors: fmt.Sprintf("error compiling regex", err)}, nil
	}

	// rewrite any mentions of the tmpdir and replace with "playground"
	// TODO: Maybe switch out "playground" with something else?
	fileEvent := Event{
		Message: regPkgName.ReplaceAllString(files[0].Source(), "pkg[\"playground\"]"),
		Kind:    "file",
	}
	events = append(events, fileEvent)

	return &ResponseEvents{Events: events}, nil
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
