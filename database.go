package shares

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
const minHashLength = 4

var (
	userPasswordKey   = []byte("password")
	userBucket        = []byte("users")
	userProfileBucket = []byte("profile")

	fileBucket  = []byte("files")
	filePathKey = []byte("path")
	fileInfoKey = []byte("info")

	hashBucket = []byte("hashes")
)

type Database struct {
	bolt *bolt.DB
}

func NewDatabase(path string) (*Database, error) {
	// i'd like to thank wessie for this function
	//  icouldn't have done it without him
	db, err := bolt.Open(path, 0600, nil)

	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(fileBucket)
		if err != nil {
			return errors.New("could not create file bucket: " + err.Error())
		}
		_, err = tx.CreateBucketIfNotExists(userBucket)
		if err != nil {
			return errors.New("could not create user bucket: " + err.Error())
		}
		_, err = tx.CreateBucketIfNotExists(hashBucket)
		if err != nil {
			return errors.New("could not create hash bucket: " + err.Error())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &Database{bolt: db}, nil
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
		bolt: db.bolt,
	}
}

// User retrieves a users data from the database
func (db *Database) User(name string) (*User, error) {
	if len(name) == 0 {
		return nil, errors.New("user does not exist")
	}

	name = strings.ToLower(name)
	var u *User

	err := db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(userBucket)
		if b == nil {
			return fmt.Errorf("user '%s' does not exist", name)
		}

		userdb := b.Bucket([]byte(name))
		if userdb == nil {
			return fmt.Errorf("user '%s' does not exist", name)
		}

		profile := userdb.Get(userProfileBucket)
		if profile == nil {
			return fmt.Errorf("user '%s' does not have a profile", name)
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
	if len(hash) == 0 {
		return nil, errors.New("file does not exist")
	}

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

func (db *Database) ShortHash(longHash []byte) string {
	var resultHash string
	var shortHash = make([]byte, hex.EncodedLen(len(longHash)))
	hex.Encode(shortHash, longHash)

	err := db.bolt.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(hashBucket)
		if b == nil {
			panic("uninitialized database, no hash bucket found")
		}

		for l := minHashLength; l < len(shortHash); l++ {
			s := shortHash[:l]

			res := b.Get(s)
			if res != nil {
				if !bytes.Equal(longHash, res) {
					// someone else has this spot in use
					continue
				}

				// existing short hash for us, use it
				resultHash = string(s)
				return nil
			}

			// found a free spot
			if err := b.Put(s, longHash); err != nil {
				return err
			}

			resultHash = string(s)
			return nil
		}

		return nil
	})

	if err != nil {
		return "error"
	}

	return resultHash
}

func (db *Database) LongHash(shortHash string) (longHash []byte) {
	db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(hashBucket)
		if b == nil {
			panic("uninitialized database, no hash bucket found")
		}

		longHash = b.Get([]byte(shortHash))
		return nil
	})

	return longHash
}
