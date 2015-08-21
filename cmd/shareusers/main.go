package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Wessie/shareserver"
)

var StorageDir string
var create bool

func init() {
	flag.StringVar(&StorageDir, "storedir", "store", "Directory to use as storage")
	flag.BoolVar(&create, "create", false, "Create user if non-existant")
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
		fmt.Println(user)
		return
	}

	if !create {
		fmt.Printf("unknown user: %s\n", name)
		return
	}

	user = state.NewUser(name)
	if !user.SetPassword("", "hello world") {
		fmt.Printf("failed to set password for new user")
		return
	}

	if err = user.Save(); err != nil {
		fmt.Printf("failed to save user after creation: %s", err)
		return
	}
}
