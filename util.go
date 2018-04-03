package shares

import (
	"crypto"
	"hash"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"

	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
)

type HashedFile interface {
	multipart.File
	hash.Hash
}

func HashReader(h crypto.Hash, f multipart.File) HashedFile {
	return hashReader{
		Hash: h.New(),
		File: f,
	}
}

type hashReader struct {
	multipart.File
	hash.Hash
}

func (h hashReader) Read(p []byte) (n int, err error) {
	n, err = h.File.Read(p)

	if _, err := h.Hash.Write(p[:n]); err != nil {
		panic("hash.Hash.Write returned an error: " + err.Error())
	}

	return
}

func removeTemporaryFile(f *os.File) {
	_ = f.Close() // no point checking error on close here
	err := os.Remove(f.Name())
	if err != nil && !os.IsNotExist(err) {
		log.Printf("failed to remove temporary file: %s\n", err)
	}
}

func Ext(path string) (ext string) {
	ext1 := filepath.Ext(path)
	if len(ext1) == len(path) {
		return ""
	}

	ext2 := filepath.Ext(path[:len(path)-len(ext1)])
	if ext2 == "" || len(ext2) > 5 {
		return ext1
	}

	ext = ext2 + ext1
	if len(ext) == len(path) {
		return ext1
	}

	return ext
}
