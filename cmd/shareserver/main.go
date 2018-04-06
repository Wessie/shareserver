package main

import (
	"crypto"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Wessie/shareserver"
)

// Configuration Values
var (
	srv http.Server
	mux = http.NewServeMux()

	TempDir       string = "store/tmp"
	DatabaseDir   string = "store/db"
	MaxFileMemory int64
	StorageDir    string = "store"
	URLPrefix     string
	Hash          crypto.Hash = crypto.SHA1
	useUserCache  bool
)

func init() {
	flag.StringVar(&srv.Addr, "addr", ":http", "address to listen on")
	flag.Int64Var(&MaxFileMemory, "maxfilememory", 16777216, "Maximum size of in-memory stored POST'd files")
	flag.StringVar(&StorageDir, "storedir", "store", "Directory to store uploaded files in")
	flag.StringVar(&URLPrefix, "prefix", "", "Prefix of URL to send to uploading client")
	flag.Var(hashChoice{&Hash}, "hash", "Hash algorithm to use for URL generation")
	flag.BoolVar(&useUserCache, "usercache", true, "Use in-memory user cache")
}

func main() {
	var err error
	args := os.Args[1:]

	if s := os.Getenv("SHARESERVER_OPTIONS"); s != "" {
		args = strings.Fields(s)
	}

	if err = flag.CommandLine.Parse(args); err != nil {
		log.Printf("failed to parse flags: %s\n", err)
		os.Exit(1)
	}

	// setup configuration fields
	// expand our user-given storage directory to be an absolute path
	if abs, err := filepath.Abs(StorageDir); err != nil {
		log.Printf("failed to expand storage directory: %s\n", err)
		os.Exit(1)
	} else {
		StorageDir = abs
	}

	TempDir = filepath.Join(StorageDir, "tmp")
	if err = os.MkdirAll(TempDir, 0770); err != nil {
		log.Printf("unable to create temporary directory: %s", err)
		os.Exit(1)
	}

	DatabaseDir = filepath.Join(StorageDir, "db")

	var state = shares.State{
		MaxFileMemory: MaxFileMemory,
		StorageDir:    StorageDir,
		TempDir:       TempDir,
		URLPrefix:     URLPrefix,
		Hash:          Hash,
	}

	if useUserCache {
		log.Println("info: using in-memory user cache")
		state.Authenticate = state.AuthenticateCache
	} else {
		log.Println("info: not using in-memory user cache")
		state.Authenticate = state.AuthenticateCrypt
	}

	// setup boltdb
	state.Database, err = shares.NewDatabase(DatabaseDir)
	if err != nil {
		log.Printf("unable to open database: %s", err)
		os.Exit(1)
	}

	// setup our server routes
	mux := http.NewServeMux()
	mux.HandleFunc("/post", state.HandlePOST)
	mux.HandleFunc("/", state.HandleGET)

	srv.Handler = mux

	log.Printf("starting shareserver on %s", srv.Addr)
	err = srv.ListenAndServe()
	if err != nil {
		log.Printf("unable to listen for http: %s", err)
		os.Exit(1)
	}
}
