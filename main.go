package main

import (
    "fmt"
    "log"
    "os"
    "os/user"
    "path/filepath"
    "regexp"
    "strconv"

    "github.com/quincy/photo_spool/spooler"
    "github.com/quincy/photo_spool/util"
)


var threads int = 1
var items chan string
var quit chan bool
var db map[string][]string = make(map[string][]string, 100)
var spoolPath string
var currentUser *user.User
var spool *spooler.Spool


/*
init sets up required initial state.
*/
func init() {
    var err error
    if currentUser, err = user.LookupId(strconv.Itoa(os.Getuid())); err != nil {
        log.Fatal(err)
    }

    // Setup paths.
    spoolPath      = filepath.Join(currentUser.HomeDir, "Desktop/spool")         // TODO these should be configurable values.
    basePhotoPath := filepath.Join(currentUser.HomeDir, "Pictures")              // TODO these should be configurable values.
    errorPath     := filepath.Join(currentUser.HomeDir, "Desktop/spool_error")   // TODO these should be configurable values.
    md5DbPath     := filepath.Join(currentUser.HomeDir, ".media_spool")          // TODO these should be configurable values.

    if err := os.MkdirAll(errorPath, 0775); err != nil {
        log.Fatal(err)
    }

    if spool, err = spooler.New(md5DbPath, basePhotoPath, errorPath); err != nil {
        log.Fatalf("Could not create a new Spool. %v\n", err)
    }
}


/*
processFiles processes each filePath read from the items channel.  For each
filePath the md5 sum is calculated, the date/time is parsed from the exif data,
a destination path is calculated, and then the file is copied to that new
destination path.  The new file name at the destination is the date and time in
YYYY-MM-DD HH:mm:ss format.

The function returns when it gets input on the quit channel.
*/
func processFiles(items chan string, quit chan bool, num int) {
    log.Println("Entering processFiles()")

    for {
        select {
        case file := <-items:
            spool.Spool(file)
        case <-quit:
            log.Println("Returning from processFiles.")
            return
        }
    }
}

/*
visit is called for each file found in a directory walk.  If the file is a
directory then it is ignored.  Otherwise the path is sent to the items channel
to be processed by the processFiles goroutine.
*/
func visit(filePath string, f os.FileInfo, err error) error {
    if !f.IsDir() {
        matched, err := regexp.MatchString("(?i:jpg$)", filePath)
        if err != nil {
            util.MoveTo(spool.ErrorPath, filePath)
        }
        if matched {
            items <- filePath
        } else {
            log.Println("Skipping non-JPEG file ", filePath)
            util.MoveTo(spool.ErrorPath, filePath)
        }
    }

    return nil
}

func main() {
    // Start the processFiles function in "threads" goroutines, this will
    // process each file that needs to be spooled when we perform the file
    // walk.
    items = make(chan string, 100)
    quit  = make(chan bool, threads)
    for i := 0; i<threads; i++ {
        go processFiles(items, quit, i)
    }

    // Start the file walk.
    err := filepath.Walk(spoolPath, visit)

    // Send the quit signal to each thread now that the file walk is complete.
    for i := 0; i<threads; i++ {
        quit<-true
    }

    if err != nil {
        fmt.Println("There was an error: ", err)
    }

    // Write out the database.
    if err := spool.Close(); err != nil {
        panic(err)
    }
}

