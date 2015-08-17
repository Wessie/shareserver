package main

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/boltdb/bolt"
)

/*
Database layout:

users:
	<username>:
		password: <password-hash>
		profile: <userprofile:json> // see UserProfile
		files: [<filehash>, ...]

files:
	<filehash>:
		path: <filepath>
		info: <fileinfo:json>

*/

// bcryptCost indicates the cost to use in bcrypt password hash generation
// NOTE: changing this is not retroactive currently, old passwords keep the
// old cost.
const bcryptCost = 13

var (
	userPasswordKey   = []byte("password")
	userBucket        = []byte("users")
	userProfileBucket = []byte("profile")

	fileBucket  = []byte("files")
	filePathKey = []byte("path")
	fileInfoKey = []byte("info")
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

func (db *Database) NewFile(sum []byte, path string) *File {
	return &File{
		Path: path,
		Hash: sum,
	}
}

// User retrieves a users data from the database
func (db *Database) User(name string) (*User, error) {
	name = strings.ToLower(name)
	var u *User

	err := db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(userBucket)
		if b == nil {
			return errors.New("user does not exist")
		}

		userdb := b.Bucket([]byte(name))
		if userdb == nil {
			return errors.New("user does not exist")
		}

		profile := userdb.Get(userProfileBucket)
		if profile == nil {
			return errors.New("user does not have a profile")
		}

		u = &User{
			Name: name,
			pwd:  userdb.Get(userPasswordKey),
			bolt: db.bolt,
		}

		return json.Unmarshal(profile, &u.UserProfile)
	})

	return u, err
}

func (db *Database) File(hash []byte) (*File, error) {
	var f *File

	err := db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucket)
		if b == nil {
			return errors.New("file does not exist")
		}

		filedb := b.Bucket(hash)
		if filedb == nil {
			return errors.New("file does not exist")
		}

		f = &File{
			Hash: hash,
			Path: string(filedb.Get(filePathKey)),
			bolt: db.bolt,
		}

		return nil
	})

	return f, err
}
