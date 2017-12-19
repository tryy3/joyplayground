package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
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
	// "strings"
	"text/template"
	"time"

	"golang.org/x/tools/imports"
)

var tmpDirFlag = flag.String("tmp-dir", "", "Path for temp folder, if not set it will use the OS temp folder.")
var joyExecutableFlag = flag.String("joy-exe", "", "Path for joy exectuable, if not set it will simply use the 'joy' command.")
var snippetStoreHost = flag.String("snippet-url", "http://localhost:5555", "URL to the snippet store.")

const maxRunTime = 2 * time.Second // Amount of time to run before timeout when executing a command

// Default golang code template
const hello = `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, playground")
}
`

var sourceDir = "." // Location of the html/static files

// HTML Templates
var indexTemplate *template.Template

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

type fmtResponse struct {
	Body  string
	Error string
}

func main() {
	flag.Parse()
	if len(os.Args) > 1 && os.Args[1] == "test" {
		// test()
		return
	}
	if source, ok := os.LookupEnv("SOURCE_DIR"); ok {
		sourceDir = source
	}

	indexTemplate = template.Must(template.ParseFiles(sourceDir + "/index.html"))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(sourceDir+"/static"))))
	http.HandleFunc("/compile", compileHandler)
	http.HandleFunc("/share", shareHandler)
	http.HandleFunc("/fmt", fmtHandler)
	http.HandleFunc("/p/", pHandler)
	http.HandleFunc("/", indexHandler)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func fmtHandler(w http.ResponseWriter, r *http.Request) {
	var (
		in  = []byte(r.FormValue("body"))
		out []byte
		err error
	)
	if r.FormValue("imports") != "" {
		out, err = imports.Process("prog.go", in, nil)
	} else {
		out, err = format.Source(in)
	}
	var resp fmtResponse
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Body = string(out)
	}
	json.NewEncoder(w).Encode(resp)
}

func pHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[2:]
	resp, err := http.Get(*snippetStoreHost + "/p/" + id)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting response from snippet store: %v", err), http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request: %v", err), http.StatusBadRequest)
		return
	}

	indexTemplate.Execute(w, &indexData{Snippet: string(body)})
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

	resp, err := http.Post(*snippetStoreHost+"/share", "application/json", bytes.NewBuffer(req))
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
	indexTemplate.Execute(w, &indexData{Snippet: hello})
}

func compileAndRun(req *Request) (*Response, error) {
	tmpDir, err := ioutil.TempDir(*tmpDirFlag, "sandbox")
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

	cmd := *joyExecutableFlag
	if cmd == "" {
		cmd = "joy"
	}

	joyCmd := exec.Command(cmd, in)
	joyCmd.Env = []string{
		"GOPATH=" + os.Getenv("GOPATH"),
		"PATH=" + os.Getenv("PATH"),
		"HOMEDRIVE=" + os.Getenv("HOMEDRIVE"),     // Windows
		"HOMEPATH=" + os.Getenv("HOMEPATH"),       // Windows
		"USERPROFILE=" + os.Getenv("USERPROFILE"), // Windows
	}

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

	// pkg := strings.Replace(tmpDir, `\`, `\\`, -1)
	regPkgName, err := regexp.Compile(`(../)+` + tmpDir)
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
