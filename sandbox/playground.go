package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const maxRunTime = 2 * time.Second
const snippetStoreHost = "http://localhost:4444"
const hello = `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, playground")
}
`

var source_dir = "."
var indexv1Template *template.Template
var indexv2Template *template.Template

type indexData struct {
	Snippet string
}

type Request struct {
	Body string
}

type Response struct {
	Errors string
	Events []Event
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		// test()
		return
	}
	if source, ok := os.LookupEnv("SOURCE_DIR"); ok {
		source_dir = source
	}

	indexv1Template = template.Must(template.ParseFiles(source_dir + "/indexv1.html"))
	indexv2Template = template.Must(template.ParseFiles(source_dir + "/indexv2.html"))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(source_dir+"/static"))))
	http.HandleFunc("/compile", compileHandler)
	http.HandleFunc("/share", shareHandler)
	http.HandleFunc("/v1/p/", pHandler)
	http.HandleFunc("/v2/p/", pHandler)
	http.HandleFunc("/", indexHandler)
	log.Fatal(http.ListenAndServe(":5555", nil))
}

func pHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[6:]
	resp, err := http.Get(snippetStoreHost + "/p/" + id)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting response from snippet store: %v", err), http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request: %v", err), http.StatusBadRequest)
		return
	}

	version := r.URL.Path[1:3]
	if version == "v1" {
		indexv1Template.Execute(w, &indexData{Snippet: string(body)})
	} else {
		indexv2Template.Execute(w, &indexData{Snippet: string(body)})
	}
}

func compileHandler(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}
	resp, err := compileAndRun(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}

func shareHandler(w http.ResponseWriter, r *http.Request) {
	req, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := http.Post(snippetStoreHost+"/share", "application/json", bytes.NewBuffer(req))
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting response from snippet store: %v", err), http.StatusBadRequest)
		return
	}

	id, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request: %v", err), http.StatusBadRequest)
		return
	}

	_, err = io.WriteString(w, string(id))
	if err != nil {
		log.Println(err)
		http.Error(w, "Server error.", http.StatusInternalServerError)
		return
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Path[1:3]
	if version == "v1" {
		indexv1Template.Execute(w, &indexData{Snippet: hello})
	} else {
		indexv2Template.Execute(w, &indexData{Snippet: hello})
	}
}

func compileAndRun(req *Request) (*Response, error) {
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
		return &Response{Errors: "package name must be main"}, nil
	}

	joyCmd := exec.Command("/root/Go/bin/joy", in)
	joyCmd.Env = []string{"GOOS=nacl", "GOARCH=amd64p32", "GOPATH=" + os.Getenv("GOPATH"), "PATH=" + os.Getenv("PATH")}

	joyRec := new(Recorder)
	joyCmd.Stdout = joyRec.File()
	joyCmd.Stderr = joyRec.Stderr()
	if err := runTimeout(joyCmd, maxRunTime); err != nil {
		if err == timeoutErr {
			return &Response{Errors: "process took too long"}, nil
		}
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, fmt.Errorf("error running sandbox: %v", err)
		}
	}

	joyEvents, err := joyRec.Events()
	if err != nil {
		return nil, fmt.Errorf("error decoding events: %v", err)
	}

	pkg := strings.Replace(tmpDir, `\`, `\\`, -1)
	// pkg = strings.Replace(tmpDir, "/", "\\/", -1)
	regPkgName, err := regexp.Compile(`(../)+` + pkg[1:])
	if err != nil {
		return nil, fmt.Errorf("error compiling regex: %v", err)
	}

	// rewrite any mentions of the tmpdir and replace with "playground"
	// TODO: Maybe switch out "playground" with something else?
	for i := 0; i < len(joyEvents); i++ {
		joyEvents[i].Message = regPkgName.ReplaceAllString(joyEvents[i].Message, "playground")
	}

	return &Response{Events: joyEvents}, nil
}

var timeoutErr = errors.New("process timed out")

func runTimeout(cmd *exec.Cmd, d time.Duration) error {
	if err := cmd.Start(); err != nil {
		fmt.Printf("%+v\n", err)
		return err
	}
	errc := make(chan error, 1)
	go func() {
		errc <- cmd.Wait()
	}()
	t := time.NewTimer(d)
	select {
	case err := <-errc:
		t.Stop()
		return err
	case <-t.C:
		cmd.Process.Kill()
		return timeoutErr
	}
}
