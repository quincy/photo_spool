package main

import (
    "bufio"
    "crypto/md5"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    "os/user"
    "path"
    "path/filepath"
    "time"

    "github.com/rwcarlsen/goexif/exif"
)


var threads int = 8
var items chan string
var quit chan bool
var db map[string][]string = make(map[string][]string, 100)
var basePhotoPath string
var spool_path string
var md5_db_path string
var currentUser *user.User


/*
init sets up required initial state.
*/
func init() {
    var err error
    if currentUser, err = user.LookupId(string(os.Getuid())); err != nil {
        log.Fatal(err)
    }

    // Setup paths.
    basePhotoPath = filepath.Join(currentUser.HomeDir, "Pictures")       // TODO these should be configurable values.
    spool_path    = filepath.Join(currentUser.HomeDir, "Desktop/spool")  // TODO these should be configurable values.
    md5_db_path   = filepath.Join(currentUser.HomeDir, ".media_spool")   // TODO these should be configurable values.

    // Read in the database of md5 sums for all previously spooled pictures.
    db = read_md5_db(md5_db_path)
}


/*
getHash calculates the md5 sum for a given filePath and returns the hex string
representation.
*/
func getHash(filePath string) string {
    h := md5.New()

    inputFile, inputError := os.Open(filePath)
    if inputError != nil {
        fmt.Printf("An error occurred while opening the input file [%s].\n", filePath)
    }
    defer inputFile.Close()

    inputReader := bufio.NewReader(inputFile)
    inputString, readerError := inputReader.ReadString('\n')
    for readerError != io.EOF {
        io.WriteString(h, inputString)
        inputString, readerError = inputReader.ReadString('\n')
    }

    sum := fmt.Sprintf("%x", h.Sum(nil))

    return sum
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
    var spoolFile string

    for {
        select {
        case spoolFile = <-items:
            fmt.Println("DEBUG ::", num, "::", spoolFile)
            // calculate an md5 sum for the file
            hash := getHash(spoolFile)

            // get the Time from the DateTimeOriginal exif tag
            dateTime := getDateTime(spoolFile)

            // calculate the path to copy the file to, including a new file name
            newPath := destinationPath(spoolFile, basePhotoPath, dateTime)

            // ensure the new path doesn't already exist
            if Exists(newPath) {
                // TODO Log the error, copy the file to a error dir, and send an e-mail.
            }

            // copy the file
            // TODO

            // add an entry to the hashmap db
            if _, exists := db[hash]; !exists {
                db[hash] = make([]string, 0, 5)
            }
            db[hash] = append(db[hash], newPath)
        case <-quit:
            return
        }
    }
}

/*
getDateTime reads the exif data from fname and returns a string representation
of the DateTimeOriginal tag.

TODO handle errors better.
*/
func getDateTime(fname string) time.Time {
    f, err := os.Open(fname)
    if err != nil {
        log.Fatal(err)
    }

    x, err := exif.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    date, _ := x.Get(exif.DateTimeOriginal)
    t       := time.Parse("2006:01:02 15:04:05", date.StringVal())

    return t
}


/*
destinationPath builds a full path where the origPath should be copied to based
on its DateTimeOriginal tag.
*/
func destinationPath(origPath, newBasePath string, t time.Time) string {
    dir   := filepath.Join(newBasePath, string(t.Year()), string(t.Month()))
    fname := filepath.Join(t.Format("2006-01-02 15:04:05"), path.Ext(origPath))

    return filepath.Join(dir, fname)
}

/*
visit is called for each file found in a directory walk.  If the file is a
directory then it is ignored.  Otherwise the path is sent to the items channel
to be processed by the processFiles goroutine.
*/
func visit(filePath string, f os.FileInfo, err error) error {
    if !f.IsDir() {
        items <- filePath
    }

    return nil
}

/*
Exists reports whether the named file or directory exists.
*/
func Exists(name string) bool {
    if _, err := os.Stat(name); err != nil {
        if os.IsNotExist(err) {
            return false
        }
    }
    return true
}


/*
read_md5_db deserializes the md5 json database from the given filePath.
*/
func read_md5_db(filePath string) map[string][]string {
    var db map[string][]string = make(map[string][]string, 100)

    if Exists(filePath) {
        json_bytes, err := ioutil.ReadFile(filePath)

        if err != nil {
            panic(err)
        }

        err = json.Unmarshal(json_bytes, &db)
    }

    return db
}


/*
write_md5_db serializes the md5 database to json and writes it to the given
file path..
*/
func write_md5_db(db map[string][]string, filePath string) {
    b, err := json.Marshal(db)

    err = ioutil.WriteFile(filePath, b, 0644)
    if err != nil { panic(err) }
}


func main() {
    // Start the processFiles function in threads goroutines, this will process
    // each file that needs to be spooled when we perform the file walk.
    items = make(chan string, 100)
    quit  = make(chan bool, threads)
    for i := 0; i<threads; i++ {
        go processFiles(items, quit, i)
    }

    // Start the file walk.
    log.Println("Starting walk...")
    err := filepath.Walk(spool_path, visit)
    log.Println("Walk complete...")

    // Send the quit signal to each thread now that the file walk is complete.
    for i := 0; i<threads; i++ {
        quit<-true
    }

    if err != nil {
        fmt.Println("There was an error: ", err)
    }

    // Write out the database.
    write_md5_db(db, md5_db_path)
}

