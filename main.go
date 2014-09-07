package main

import (
    "fmt"
    "log"
    "os"
    "os/user"
    "path/filepath"
    "regexp"

    "github.com/quincy/configo"
    "github.com/quincy/photo_spool/spooler"
    "github.com/quincy/photo_spool/util"
)

var db map[string][]string = make(map[string][]string, 100)
var currentUser *user.User
var spool *spooler.Spool
var spoolPath string
var basePhotoPath string
var errorPath string
var md5DbPath string
var noop bool

/*
setup configuration options
*/
func init() {
    var err error
    if currentUser, err = user.Current(); err != nil {
        log.Println("Error looking up user id.")
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

    if spool, err = spooler.New(md5DbPath, basePhotoPath, errorPath, noop); err != nil {
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
        pattern := "(?i:jpe?g$)"
        matched, err := regexp.MatchString(pattern, filePath)
        if err != nil {
            log.Fatalf("Error compiling regular expression '%s'.  %v", pattern, err)
        }

        if matched {
            spoolError := spool.Spool(filePath)
            if spoolError != nil {
                log.Println(spoolError)
            }

            parent := filepath.Dir(filePath)
            if util.DirIsEmpty(parent) {
                log.Printf("Pruning empty directory [%s].\n", parent)
                if noop {
                    log.Println("DRY RUN Skipping delete directory.")
                } else {
                    os.Remove(parent)
                }
            }
        } else {
            log.Printf("Found unhandled file type [%s].  Moving the file to %s.\n", filePath, spool.ErrorPath)
            if noop {
                log.Println("DRY RUN Skipping move file.")
            } else {
                util.MoveTo(spool.ErrorPath, filePath)
            }
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
