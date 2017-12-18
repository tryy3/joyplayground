// Copied from https://github.com/gopherjs/snippet-store
// snippet-store is a server for storing GopherJS Playground snippets.
//
// It uses the same mapping scheme as the Go Playground, and when a snippet isn't found locally,
// it defers to fetching it from the Go Playground. This effectively augments our world of available
// snippets with that of the Go Playground.
//
// Newly shared snippets are stored locally in the specified folder and take precedence.
package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/webdav"
)

var storageDirFlag = flag.String("storage-dir", "", "Storage dir for snippets; if empty, a volatile in-memory store is used.")
var httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")

// var allowOriginFlag = flag.String("allow-origin", "http://www.gopherjs.org", "Access-Control-Allow-Origin header value.")

const maxSnippetSizeBytes = 1024 * 1024

func pHandler(w http.ResponseWriter, req *http.Request) {
	// w.Header().Set("Access-Control-Allow-Origin", *allowOriginFlag)

	if req.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method should be GET.", http.StatusMethodNotAllowed)
		return
	}

	id := req.URL.Path[len("/p/"):]
	err := validateID(id)
	if err != nil {
		http.Error(w, "Unexpected id format.", http.StatusBadRequest)
		return
	}

	var snippet io.Reader
	if rc, err := getSnippetFromLocalStore(req.Context(), id); err == nil { // Check if we have the snippet locally first.
		defer rc.Close()
		snippet = rc
	} else if rc, err = getSnippetFromGoPlayground(req.Context(), id); err == nil { // If not found locally, try the Go Playground.
		defer rc.Close()
		snippet = rc
	}

	if snippet == nil {
		// Snippet not found.
		http.Error(w, "Snippet not found.", http.StatusNotFound)
		return
	}

	_, err = io.Copy(w, snippet)
	if err != nil {
		log.Println(err)
		http.Error(w, "Server error.", http.StatusInternalServerError)
		return
	}
}

func shareHandler(w http.ResponseWriter, req *http.Request) {
	// w.Header().Set("Access-Control-Allow-Origin", *allowOriginFlag)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // Needed for Safari.

	if req.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method should be POST.", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(http.MaxBytesReader(w, req.Body, maxSnippetSizeBytes))
	if err != nil {
		log.Println(err)
		http.Error(w, "Server error.", http.StatusInternalServerError)
		return
	}

	id, err := storeSnippet(req.Context(), body)
	if err != nil {
		http.Error(w, "Server error.", http.StatusInternalServerError)
		return
	}

	_, err = io.WriteString(w, id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Server error.", http.StatusInternalServerError)
		return
	}
}

func main() {
	flag.Parse()

	switch *storageDirFlag {
	default:
		err := os.MkdirAll(*storageDirFlag, 0755)
		if err != nil {
			log.Fatalf("Error creating directory %q: %v.\n", *storageDirFlag, err)
		}
		localStore = webdav.Dir(*storageDirFlag)
	case "":
		localStore = webdav.NewMemFS()
	}

	http.HandleFunc("/p/", pHandler)        // "/p/{{.SnippetId}}", serve snippet by id.
	http.HandleFunc("/share", shareHandler) // "/share", save snippet and return its id.

	log.Println("Started.")
	log.Println("Server running on " + *httpFlag)

	err := http.ListenAndServe(*httpFlag, nil)
	if err != nil {
		log.Fatalln("ListenAndServe:", err)
	}
}