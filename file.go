package main

import (
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"
)

type File struct {
	// Hash generated from file contents
	Hash []byte
	// Path is the filepath to the file on-disk
	Path string

	FileInfo

	// bolt is the boltdb associated with this file
	bolt *bolt.DB
}

type FileInfo struct {
	// Filename is the filename as given by uploader
	Filename string
}

func (f File) Save() error {
	if len(f.Hash) == 0 {
		return errors.New("invalid filehash: empty")
	}
	if len(f.Path) == 0 {
		return errors.New("invalid filepath: empty")
	}
	if f.bolt == nil {
		return errors.New("invalid file instance: no database")
	}

	return f.bolt.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(fileBucket)
		if err != nil {
			return err
		}

		filedb, err := b.CreateBucketIfNotExists(f.Hash)
		if err != nil {
			return err
		}

		// Store the files on-disk path
		if err := filedb.Put(filePathKey, []byte(f.Path)); err != nil {
			return err
		}

		// Store the files info
		fileInfo, err := json.Marshal(f.FileInfo)
		if err != nil {
			return err
		}

		if err := filedb.Put(fileInfoKey, fileInfo); err != nil {
			return err
		}

		return nil
	})
}
