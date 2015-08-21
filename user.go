package shares

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/boltdb/bolt"
	"golang.org/x/crypto/bcrypt"
)

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
