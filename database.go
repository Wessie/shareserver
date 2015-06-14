package main

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/boltdb/bolt"
)

type Database struct {
	bolt *bolt.DB
}

func (db *Database) NewUser(name string) *User {
	return &User{
		Name: strings.ToLower(name),
		bolt: db.bolt,
	}
}

func (db *Database) User(name string) (*User, error) {
	name = strings.ToLower(name)
	var u *User

	err := db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return errors.New("user does not exist")
		}

		userdb := b.Bucket([]byte(name))
		if userdb == nil {
			return errors.New("user does not exist")
		}

		profile := userdb.Get([]byte("profile"))
		if profile == nil {
			return errors.New("user does not have a profile")
		}

		u = &User{
			Name: name,
			bolt: db.bolt,
		}

		u.RawProfile = append(u.RawProfile, profile...)
		return json.Unmarshal(profile, u)
	})

	return u, err
}

func (db *Database) File(hash string) (*File, error) {
	var f *File

	err := db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		if b == nil {
			return errors.New("file does not exist")
		}

		fileInfo := b.Get([]byte(hash))
		if fileInfo == nil {
			return errors.New("file does not exist")
		}

		f = &File{
			Hash: hash,
			bolt: db.bolt,
		}

		f.RawInfo = append(f.RawInfo, fileInfo...)
		return json.Unmarshal(fileInfo, f)
	})

	return f, err
}

type User struct {
	Name string
	Hash string

	RawProfile []byte
	bolt       *bolt.DB
}

type File struct {
	Hash string

	RawInfo []byte
	bolt    *bolt.DB
}
