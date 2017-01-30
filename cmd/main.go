package main

import (
	"fmt"
	"github.com/quincy/configo"
	"github.com/quincy/photo_spool/spooler"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

var db map[string][]string = make(map[string][]string, 100) // FIXME unused variable
var currentUser *user.User
var spoolPath string
var basePhotoPath string
var errorPath string
var md5DbPath string
var noop bool

// setup configuration options
func init() {
	var err error
	if currentUser, err = user.Current(); err != nil {
		log.Fatal(err)
	}

	configo.StringVar(&spoolPath, "spooldir", filepath.Join(currentUser.HomeDir, "spool"), "The directory to look in for new pictures to spool.")
	configo.StringVar(&basePhotoPath, "photodir", filepath.Join(currentUser.HomeDir, "Pictures"), "The directory that new pictures will be copied to as they are spooled.")
	configo.StringVar(&errorPath, "errordir", filepath.Join(currentUser.HomeDir, "spool_error"), "The directory to copy pictures to when an error occurs.")
	configo.StringVar(&md5DbPath, "dbdir", filepath.Join(currentUser.HomeDir, ".photo-spool.db"), "The full path to the photo database file.")
	configo.BoolFlagVar(&noop, "dryrun", false, "If set the program has no effect but prints what would have happened.")

	if err = configo.Parse(); err != nil {
		panic(err)
	}

	if err := os.MkdirAll(errorPath, 0775); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(basePhotoPath, 0775); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(spoolPath, 0775); err != nil {
		log.Fatal(err)
	}
}

func main() {
	spool, err := spooler.New(md5DbPath, basePhotoPath, errorPath, noop)
	if err != nil {
		log.Fatalf("Could not create a new Spool. %v\n", err)
	}

	// Start the file walk.
	err = filepath.Walk(spoolPath, spooler.GetWalkFunc(spool, spoolPath, noop))

	if err != nil {
		fmt.Println("There was an error: ", err)
	}

	// Write out the database.
	if err := spool.Close(); err != nil {
		panic(err)
	}
}
