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
	"sync"
)

type State struct {
	*Database
	cache        userCache
	Authenticate func(*User, string) bool

	MaxFileMemory int64
	StorageDir    string
	TempDir       string
	URLPrefix     string
	Hash          crypto.Hash
}

type userCache struct {
	sync.RWMutex
	c map[string]string
}

// AuthenticateCache checks if user and password match together, and uses an
// in-memory cache after a success
func (s *State) AuthenticateCache(user *User, pwd string) bool {
	s.cache.RLock()
	cachePwd, ok := s.cache.c[user.Name]
	s.cache.RUnlock()
	if ok {
		log.Println("authenticate: using fast path")
		// fast path, from cache
		return cachePwd == pwd
	}

	log.Println("authenticate: using slow path")

	// slow path, we have to bcrypt compare and cache it
	if !user.ComparePassword(pwd) {
		return false
	}

	s.cache.Lock()
	if s.cache.c == nil {
		s.cache.c = make(map[string]string, 5)
	}
	s.cache.c[user.Name] = pwd
	s.cache.Unlock()

	return true
}

// AuthenticateCrypt checks if user and password match together, and always
// uses a more expensive comparison
func (s *State) AuthenticateCrypt(user *User, pwd string) bool {
	return user.ComparePassword(pwd)
}

// HandlePOST handles sharex uploads
//
// We use the 'file' formfield as file upload field
func (s *State) HandlePOST(rw http.ResponseWriter, r *http.Request) {
	// TODO: Handle big requests
	err := r.ParseMultipartForm(s.MaxFileMemory)
	if err != nil {
		// TODO: error
		log.Printf("error: failed parsing POST form: %s\n", err)
		return
	}

	username := r.FormValue("user")
	password := r.FormValue("pass")

	user, err := s.Database.User(username)
	if err != nil {
		log.Printf("error: authenticate: invalid user: %s\n", err)
		return
	}

	if !s.Authenticate(user, password) {
		log.Printf("error: authenticate: failed authenticate user: %s\n", user.Name)
		return
	}

	formFile, fileHeader, err := r.FormFile("file")
	if err != nil {
		// TODO: error
		log.Printf("error: failed parsing form (formfile): %s\n", err)
		return
	}
	defer formFile.Close()

	// Wrap our file to generate a hash while we're saving it.
	var file = HashReader(s.Hash, formFile)

	tempFile, err := ioutil.TempFile(s.TempDir, "shares")
	if err != nil {
		log.Printf("error: failed to open temporary file: %s\n", err)
		return
	}
	defer removeTemporaryFile(tempFile)

	_, err = io.Copy(tempFile, file)
	if err != nil {
		log.Printf("error: failed to copy contents to temporary file: %s\n", err)
		return
	}

	sum := file.Sum(nil)
	newPath := filepath.Join(s.StorageDir, fmt.Sprintf("%x", sum))

	log.Printf("info: storing file as %s\n", newPath)
	log.Printf("info: %s %x\n", tempFile.Name(), file.Sum(nil))

	err = os.MkdirAll(filepath.Dir(newPath), 0755)
	if err != nil {
		log.Printf("error: failed to create directories: %s\n", err)
		return
	}

	err = os.Rename(tempFile.Name(), newPath)
	if err != nil && !os.IsExist(err) {
		log.Printf("error: failed to rename temporary file: %s\n", err)
		return
	}

	dbFile := s.NewFile(sum, newPath)
	dbFile.Filename = fileHeader.Filename
	if err = dbFile.Save(); err != nil {
		log.Printf("error: failed to save file information: %s\n", err)
		// TODO: cleanup the file we just saved?
		return
	}

	log.Printf("succes: %s --> %s\n", fileHeader.Filename, newPath)

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
