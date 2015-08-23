package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Wessie/shareserver"
	"github.com/howeyc/gopass"
)

var StorageDir string
var create bool
var chpasswd bool

func init() {
	flag.StringVar(&StorageDir, "storedir", "store", "Directory to use as storage")
	flag.BoolVar(&create, "create", false, "Create user if non-existant")
	flag.BoolVar(&chpasswd, "pass", false, "Set password")
}

func main() {
	var err error
	flag.Parse()

	state := shares.State{
		StorageDir: StorageDir,
	}

	DatabaseDir := filepath.Join(StorageDir, "db")

	state.Database, err = shares.NewDatabase(DatabaseDir)
	if err != nil {
		log.Printf("unable to open database: %s", err)
		os.Exit(1)
	}

	name := flag.Arg(0)
	if name == "" {
		log.Printf("no user given")
		os.Exit(1)
	}

	user, err := state.User(name)
	if err == nil {
		fmt.Printf("Loaded user %s\n", name)
	} else if create == true {
		user = state.NewUser(name)
		fmt.Printf("created new user: %s\n", name)
	} else {
		fmt.Printf("unknown user: %s\n", name)
		return
	}

	if chpasswd == true {
		fmt.Printf("Enter current password:")
		curPass := gopass.GetPasswd()
		fmt.Printf("Enter new password:")
		newPass := gopass.GetPasswd()
		fmt.Printf("Enter new password again:")
		if another := gopass.GetPasswd() ; string(newPass) != string(another) {
			fmt.Printf("new passwords did not match\n")
			return
		}
		if !user.SetPassword(string(curPass), string(newPass)) {
			fmt.Printf("failed to set password for new user")
			return
		}
		fmt.Printf("successfully set new password for user %s\n", name)
	}

	if err = user.Save(); err != nil {
		fmt.Printf("failed to save user after creation: %s\n", err)
		return
	}
}
