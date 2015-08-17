package main

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/boltdb/bolt"
	"golang.org/x/crypto/bcrypt"
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

// User represents a single user in the database
type User struct {
	Name string
	pwd  []byte

	UserProfile

	// bolt is the boltdb associated with this user
	bolt *bolt.DB
}

// ComparePassword compares pwd with the saved password
func (u User) ComparePassword(pwd string) bool {
	// empty known password is always a rejection.
	if len(u.pwd) == 0 || len(pwd) == 0 {
		return false
	}

	err := bcrypt.CompareHashAndPassword(u.pwd, []byte(pwd))

	return err == nil
}

// SetPassword sets a new password for user.
func (u *User) SetPassword(current, new string) bool {
	// empty current password means we're setting our first password, this
	// means we can skip the compare.
	if len(u.pwd) != 0 && !u.ComparePassword(current) {
		return false
	}

	bp, err := bcrypt.GenerateFromPassword([]byte(new), bcryptCost)
	if err != nil {
		log.Printf("critical: failed to generate bcrypt hash: %s", err)
		return false
	}

	u.pwd = bp
	return true
}

func (u User) Save() error {
	if len(u.pwd) == 0 {
		return errors.New("invalid password: empty")
	}
	if len(u.Name) == 0 {
		return errors.New("invalid username: empty")
	}
	if u.bolt == nil {
		return errors.New("invalid user instance: no database")
	}

	return u.bolt.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(userBucket)
		if err != nil {
			// TODO: error clarification
			return err
		}

		userdb, err := b.CreateBucketIfNotExists([]byte(u.Name))
		if err != nil {
			// TODO: error clarification
			return err
		}

		// Store the users password
		if err := userdb.Put(userPasswordKey, u.pwd); err != nil {
			return errors.New("failed to store password: " + err.Error())
		}

		// Store the users profile
		userProfile, err := json.Marshal(u.UserProfile)
		if err != nil {
			return errors.New("failed to marshal userprofile: " + err.Error())
		}

		if err := userdb.Put(userProfileBucket, userProfile); err != nil {
			return errors.New("failed to store userprofile: " + err.Error())
		}

		return nil
	})
}

// UserProfile contains non-critical information about a User
type UserProfile struct{}

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
