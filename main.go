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
    "regexp"
    "strconv"
    "time"

    "github.com/rwcarlsen/goexif/exif"
)


var threads int = 1
var items chan string
var quit chan bool
var db map[string][]string = make(map[string][]string, 100)
var basePhotoPath string
var errorPath string
var spoolPath string
var md5DbPath string
var currentUser *user.User


/*
init sets up required initial state.
*/
func init() {
    log.Println("Entering init()")
    var err error
    if currentUser, err = user.LookupId(string(os.Getuid())); err != nil {
        log.Fatal(err)
    }

    log.Println("currentUser = ", currentUser)

    // Setup paths.
    basePhotoPath = filepath.Join(currentUser.HomeDir, "Pictures")              // TODO these should be configurable values.
    errorPath     = filepath.Join(currentUser.HomeDir, "Desktop/spool_error")   // TODO these should be configurable values.
    spoolPath     = filepath.Join(currentUser.HomeDir, "Desktop/spool")         // TODO these should be configurable values.
    md5DbPath     = filepath.Join(currentUser.HomeDir, ".media_spool")          // TODO these should be configurable values.


    log.Println("basePhotoPath = ", basePhotoPath)
    log.Println("errorPath     = ", errorPath)
    log.Println("spoolPath     = ", spoolPath)
    log.Println("md5DbPath     = ", md5DbPath)

    // Read in the database of md5 sums for all previously spooled pictures.
    db = readMd5Db(md5DbPath)

    log.Println("Returning from init()")
}


/*
getHash calculates the md5 sum for a given filePath and returns the hex string
representation.
*/
func getHash(filePath string) string {
    log.Println("Entering getHash(", filePath, ")")
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

    log.Println("Returning ", sum, " from getHash(", filePath, ")")
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
    log.Println("Entering processFiles()")
    var spoolFile string

    for {
        select {
        case spoolFile = <-items:
            log.Println(num, "::", spoolFile)
            // calculate an md5 sum for the file
            hash := getHash(spoolFile)

            // get the Time from the DateTimeOriginal exif tag
            dateTime := getDateTime(spoolFile)

            // calculate the path to copy the file to, including a new file name
            newPath := destinationPath(spoolFile, basePhotoPath, dateTime)

            // ensure the new path doesn't already exist
            if Exists(newPath) {
                moveFileToErrorPath(spoolFile)
                log.Fatal("A file with that name already exists at the destination.")  // TODO This logging sucks.
                // TODO send an e-mail.
            }

            // move the file to its new home
            if err := mv(newPath, spoolFile); err != nil {
                log.Fatal(err)
            }

            // add an entry to the hashmap db
            if _, exists := db[hash]; !exists {
                db[hash] = make([]string, 0, 5)
            }
            db[hash] = append(db[hash], newPath)
        case <-quit:
            log.Println("Returning from processFiles.")
            return
        }
    }
}

/*
getDateTime reads the exif data from fname and returns a string representation
of the DateTimeOriginal tag.
*/
func getDateTime(fname string) time.Time {
    log.Println("Entering getDateTime(", fname, ")")
    f, err := os.Open(fname)
    if err != nil {
        mv(fname, filepath.Join(errorPath, fname))
        log.Fatal(err)
    }

    log.Println("Decoding exif data for ", fname)
    x, err := exif.Decode(f)
    if err != nil {
        mv(fname, filepath.Join(errorPath, fname))
        log.Fatal(err)
    }

    date, _ := x.Get(exif.DateTimeOriginal)
    log.Println("Setting DateTimeOriginal to ", date.StringVal(), " on ", fname)
    t, err  := time.Parse("2006:01:02 15:04:05", date.StringVal())
    if err != nil {
        mv(fname, filepath.Join(errorPath, fname))
        log.Fatal(err)
    }

    log.Printf("Returning %v from getDateTime()", t)
    return t
}

/*
moveFileToErrorPath moves the file at fname to the errorPath directory.
*/
func moveFileToErrorPath(fname string) error {
    log.Println("Entering moveFileToErrorPath(", fname, ")")
    dst := filepath.Join(errorPath, fname)
    if err:= mv(dst, fname); err != nil {
        return err
    }

    return nil
}

/*
mv copies the file at src to the new file at dst.
Credit: https://gist.github.com/elazarl/5507969
*/
func mv(dst, src string) error {
    log.Printf("Entering mv(%s, %s)", dst, src)
    s, err := os.Open(src)
    if err != nil {
        return err
    }
    // no need to check errors on read only file, we already got everything
    // we need from the filesystem, so nothing can go wrong now.
    defer s.Close()
    d, err := os.Create(dst)
    if err != nil {
        return err
    }
    if _, err := io.Copy(d, s); err != nil {
        d.Close()
        return err
    }
    if err := d.Close(); err != nil {
        return err
    }

    if err:= os.Remove(src); err != nil {
        return err
    }

    log.Println("Returning from mv()")
    return nil
}

/*
destinationPath builds a full path where the origPath should be copied to based
on its DateTimeOriginal tag.
*/
func destinationPath(origPath, newBasePath string, t time.Time) string {
    log.Printf("Entering destinationPath(%v, %v, %v)", origPath, newBasePath, t)
    dir   := filepath.Join(newBasePath, string(t.Year()), string(t.Month()))
    fname := filepath.Join(t.Format("2006-01-02 15:04:05"), path.Ext(origPath))

    log.Printf("Returning %v from destinationPath()", filepath.Join(dir, fname))
    return filepath.Join(dir, fname)
}

/*
visit is called for each file found in a directory walk.  If the file is a
directory then it is ignored.  Otherwise the path is sent to the items channel
to be processed by the processFiles goroutine.
*/
func visit(filePath string, f os.FileInfo, err error) error {
    log.Println("Entering visit()")
    if !f.IsDir() {
        matched, err := regexp.MatchString("(?i:jpg$)", filePath)
        if err != nil {
            moveFileToErrorPath(filePath)
        }
        if matched {
            items <- filePath
        } else {
            log.Println("Skipping non-JPEG file ", filePath)
        }
    }

    log.Println("Returning from visit.")
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
readMd5Db deserializes the md5 json database from the given filePath.
*/
func readMd5Db(filePath string) map[string][]string {
    log.Println("Entering readMd5Db(", filePath, ")")
    var db map[string][]string = make(map[string][]string, 100)

    if Exists(filePath) {
        json_bytes, err := ioutil.ReadFile(filePath)

        if err != nil {
            panic(err)
        }

        err = json.Unmarshal(json_bytes, &db)
    }

    log.Printf("Returning from readMd5Db")
    return db
}


/*
writeMd5Db serializes the md5 database to json and writes it to the given
file path..
*/
func writeMd5Db(db map[string][]string, filePath string) {
    log.Println("Entering writeMd5Db()")
    b, err := json.Marshal(db)

    err = ioutil.WriteFile(filePath, b, 0644)
    if err != nil {
        panic(err)
    }

    log.Println("Returning from writeMd5Db()")
}


func main() {
    log.Println("Entering main()")
    // Start the processFiles function in threads goroutines, this will process
    // each file that needs to be spooled when we perform the file walk.
    items = make(chan string, 100)
    quit  = make(chan bool, threads)
    for i := 0; i<threads; i++ {
        go processFiles(items, quit, i)
    }

    // Start the file walk.
    log.Println("Starting walk...")
    err := filepath.Walk(spoolPath, visit)
    log.Println("Walk complete...")

    // Send the quit signal to each thread now that the file walk is complete.
    for i := 0; i<threads; i++ {
        quit<-true
    }

    if err != nil {
        fmt.Println("There was an error: ", err)
    }

    // Write out the database.
    writeMd5Db(db, md5DbPath)
    log.Println("Returning from main()")
}

