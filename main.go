package main

import (
    "fmt"
    "log"
    "os"
    "os/user"
    "path/filepath"
    "regexp"

    "github.com/quincy/photo_spool/spooler"
    "github.com/quincy/photo_spool/util"
)

var db map[string][]string = make(map[string][]string, 100)
var spoolPath string
var currentUser *user.User
var spool *spooler.Spool

/*
init sets up required initial state.
*/
func init() {
    var err error
    if currentUser, err = user.Current(); err != nil {
        log.Fatal(err)
    }

    // Setup paths.
    //    spoolPath = filepath.Join(currentUser.HomeDir, "spool")            // TODO these should be configurable values.
    //    basePhotoPath := filepath.Join(currentUser.HomeDir, "Pictures")    // TODO these should be configurable values.
    //    errorPath := filepath.Join(currentUser.HomeDir, "spool_error")     // TODO these should be configurable values.
    //    md5DbPath := filepath.Join(currentUser.HomeDir, ".photo-spool.db") // TODO these should be configurable values.
    spoolPath = filepath.Join("/media/sf_C_DRIVE/Users/quincy/spool")            // TODO these should be configurable values.
    basePhotoPath := filepath.Join("/media/sf_C_DRIVE/Users/quincy/Pictures")    // TODO these should be configurable values.
    errorPath := filepath.Join("/media/sf_C_DRIVE/Users/quincy/spool_error")     // TODO these should be configurable values.
    md5DbPath := filepath.Join("/media/sf_C_DRIVE/Users/quincy/.photo-spool.db") // TODO these should be configurable values.

    if err := os.MkdirAll(errorPath, 0775); err != nil {
        log.Fatal(err)
    }

    if err := os.MkdirAll(basePhotoPath, 0775); err != nil {
        log.Fatal(err)
    }

    if err := os.MkdirAll(spoolPath, 0775); err != nil {
        log.Fatal(err)
    }

    if spool, err = spooler.New(md5DbPath, basePhotoPath, errorPath); err != nil {
        log.Fatalf("Could not create a new Spool. %v\n", err)
    }
}

/*
visit is called for each file found in a directory walk.  If the file is a
directory then it is ignored.  Otherwise the path is sent to the items channel
to be processed by the processFiles goroutine.
*/
func visit(filePath string, f os.FileInfo, err error) error {
    log.Println("Visiting", filePath)

    if !f.IsDir() {
        matched, err := regexp.MatchString("(?i:jpe?g$)", filePath)
        if err != nil {
            util.MoveTo(spool.ErrorPath, filePath)
        }
        if matched {
            spoolError := spool.Spool(filePath)
            if spoolError != nil {
                log.Println(spoolError)
            }

            parent := filepath.Dir(filePath)
            if util.DirIsEmpty(parent) {
                // TODO Prune the parent
            }
        } else {
            log.Println("Skipping non-JPEG file ", filePath)
            util.MoveTo(spool.ErrorPath, filePath)
        }
    }

    return nil
}

func main() {
    // Start the file walk.
    err := filepath.Walk(spoolPath, visit)

    if err != nil {
        fmt.Println("There was an error: ", err)
    }

    // Write out the database.
    if err := spool.Close(); err != nil {
        panic(err)
    }
}
