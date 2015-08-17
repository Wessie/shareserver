package main

import (
	"crypto"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var MaxFileMemory int64
var StorageDir string
var Hash crypto.Hash = crypto.SHA1

func init() {
	flag.Int64Var(&MaxFileMemory, "maxfilememory", 16777216, "Maximum size of in-memory stored POST'd files")
	flag.StringVar(&StorageDir, "storedir", "store", "Directory to store uploaded files in")
	flag.Var(hashChoice{&Hash}, "hash", "Hash algorithm to use for URL generation")
}

func (s *State) HandlePOST(rw http.ResponseWriter, r *http.Request) {
	// TODO: Handle big requests
	err := r.ParseMultipartForm(MaxFileMemory)
	if err != nil {
		// TODO: error
		log.Printf("failed parsing POST form: %s\n", err)
		return
	}

	formFile, fileHeader, err := r.FormFile("file")
	if err != nil {
		// TODO: error
		log.Printf("failed parsing POST form (formfile): %s\n", err)
		return
	}
	defer formFile.Close()

	// Wrap our file to generate a hash while we're saving it.
	var file HashedFile = HashReader(Hash, formFile)

	tempFile, err := ioutil.TempFile(TempDir, "shares")
	if err != nil {
		log.Printf("failed to open temporary file: %s\n", err)
		return
	}
	defer removeTemporaryFile(tempFile)

	_, err = io.Copy(tempFile, file)
	if err != nil {
		log.Printf("failed to copy contents to temporary file: %s\n", err)
		return
	}

	sum := file.Sum(nil)
	newPath := filepath.Join(StorageDir, fmt.Sprintf("%x", sum))

	log.Printf("storing file as %s\n", newPath)
	log.Printf("%s %x\n", tempFile.Name(), file.Sum(nil))

	err = os.MkdirAll(filepath.Dir(newPath), 0755)
	if err != nil {
		log.Printf("failed to create directories: %s\n", err)
		return
	}

	err = os.Rename(tempFile.Name(), newPath)
	if err != nil && !os.IsExist(err) {
		log.Printf("failed to rename temporary file: %s\n", err)
		return
	}

	dbFile := s.NewFile(sum, newPath)
	if err = dbFile.Save(); err != nil {
		log.Printf("failed to save file information: %s\n", err)
		return
	}

	log.Printf("succeeded: %s --> %s\n", fileHeader.Filename, newPath)
}
