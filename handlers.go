package shares

import (
	"crypto"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type State struct {
	*Database

	MaxFileMemory int64
	StorageDir    string
	TempDir       string
	URLPrefix     string
	Hash          crypto.Hash
}

// HandlePOST handles sharex uploads
//
// We use the 'file' formfield as file upload field
func (s *State) HandlePOST(rw http.ResponseWriter, r *http.Request) {
	// TODO: Handle big requests
	err := r.ParseMultipartForm(s.MaxFileMemory)
	if err != nil {
		// TODO: error
		log.Printf("failed parsing POST form: %s\n", err)
		return
	}

	username := r.FormValue("user")
	password := r.FormValue("pass")

	user, err := s.Database.User(username)
	if err != nil {
		log.Printf("invalid user: %s\n", err)
		return
	}

	if !user.ComparePassword(password) {
		log.Printf("incorrect password\n")
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
	var file HashedFile = HashReader(s.Hash, formFile)

	tempFile, err := ioutil.TempFile(s.TempDir, "shares")
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
	newPath := filepath.Join(s.StorageDir, fmt.Sprintf("%x", sum))

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
	dbFile.Filename = fileHeader.Filename
	if err = dbFile.Save(); err != nil {
		log.Printf("failed to save file information: %s\n", err)
		return
	}

	log.Printf("succeeded: %s --> %s\n", fileHeader.Filename, newPath)

	fmt.Fprintf(rw, "%s%s%s",
		s.URLPrefix,
		s.ShortHash(sum),
		Ext(fileHeader.Filename),
	)
}

func (s *State) HandleGET(rw http.ResponseWriter, r *http.Request) {
	extSize := len(Ext(r.URL.Path))

	path := r.URL.Path[1 : len(r.URL.Path)-extSize]
	hash := s.LongHash(path)
	if hash == nil {
		log.Printf("no such file: %s", r.URL.Path)
		return
	}

	f, err := s.File(hash)
	if err != nil {
		log.Printf("no such file: %s: %s", r.URL.Path, err)
		return
	}

	http.ServeFile(rw, r, f.Path)
}
