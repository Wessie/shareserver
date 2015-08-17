package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Configuration Values
var (
	TempDir string = "store/tmp"
)

var srv http.Server
var mux = http.NewServeMux()

func init() {
	flag.StringVar(&srv.Addr, "addr", ":http", "address to listen on")
}

func main() {
	var err error
	flag.Parse()

	// setup configuration fields
	TempDir, err = filepath.Abs(filepath.Join(StorageDir, "tmp"))
	if err != nil {
		log.Printf("unable to generate temporary directory: %s", err)
		os.Exit(1)
	}

	if err = os.MkdirAll(TempDir, 0770); err != nil {
		log.Printf("unable to create temporary directory: %s", err)
		os.Exit(1)
	}

	var state State

	// expand our user-given storage directory to be an absolute path
	if abs, err := filepath.Abs(StorageDir); err != nil {
		log.Printf("failed to expand storage directory: %s\n", err)
		os.Exit(1)
	} else {
		StorageDir = abs
	}

	// setup our server routes
	mux := http.NewServeMux()
	mux.HandleFunc("/post", state.HandlePOST)

	srv.Handler = mux

	err = srv.ListenAndServe()
	if err != nil {
		log.Printf("unable to listen for http: %s", err)
		os.Exit(1)
	}
}

type State struct {
	*Database
}
