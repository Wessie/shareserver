package main

import (
	"crypto"
	"fmt"
	"hash"
	"log"
	"mime/multipart"
	"os"
	"strings"

	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
)

type hashChoice struct {
	*crypto.Hash
}

func (h hashChoice) Set(value string) error {
	var hh crypto.Hash

	switch strings.ToLower(value) {
	case "sha1":
		hh = crypto.SHA1
	case "sha256":
		hh = crypto.SHA256
	case "sha512":
		hh = crypto.SHA512
	default:
		return fmt.Errorf("unsupported hash function: %s", value)
	}

	*h.Hash = hh
	return nil
}

func (h hashChoice) String() string {
	return "sha1"
}

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
