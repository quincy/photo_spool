package spooler

import (
    "bufio"
    "crypto/md5"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    "path"
    "path/filepath"
    "strconv"
    "time"

    "github.com/rwcarlsen/goexif/exif"
    "github.com/quincy/photo_spool/util"
)

type Spool struct {
    Destination string
    dbPath string
    ErrorPath string
    db map[string][]string
}

// New creates and returns a new *Spool.
func New(dbPath, destination, errorPath string) (*Spool, error) {
    sp := new(Spool)

    if !util.Exists(destination) {
        if _, err := os.Create(destination); err != nil {
            return nil, err
        }
    }
    sp.Destination = destination

    if !util.Exists(errorPath) {
        if _, err := os.Create(errorPath); err != nil {
            return nil, err
        }
    }
    sp.ErrorPath = errorPath

    sp.dbPath = dbPath
    if err := sp.readDatabase(); err != nil {
        return nil, err
    }

    return sp, nil
}

// readDatabase deserializes the md5 json database from the given filePath.
func (sp *Spool) readDatabase() error {
    var db map[string][]string

    if util.Exists(sp.dbPath) {
        json_bytes, err := ioutil.ReadFile(sp.dbPath)

        if err != nil {
            return err
        }

        err = json.Unmarshal(json_bytes, &db)
    } else {
        db = make(map[string][]string)
    }

    sp.db = db
    return nil
}

// writeDatabase serializes the md5 database to json and writes it to disk.
func (sp *Spool) writeDatabase() error {
    b, err := json.Marshal(sp.db)
    if err != nil {
        return err
    }

    err = ioutil.WriteFile(sp.dbPath, b, 0644)
    return err
}

func (sp *Spool) Close() error {
    return sp.writeDatabase()
}

func (sp *Spool) Spool(file string) error {
    // calculate an md5 sum for the file
    hash := getHash(file)

    // get the Time from the DateTimeOriginal exif tag
    dateTime, err := getDateTime(file)
    if err != nil {
        log.Printf("Could not read the DateTimeOriginal tag. %v", err)
        log.Printf("Moving %s to %s.\n", file, sp.ErrorPath)
        if mverr := util.MoveTo(sp.ErrorPath, file); mverr != nil {
            log.Fatal(mverr)
        }
        return err
    }

    // calculate the path to copy the file to, including a new file name
    newPath := sp.getDestination(file, sp.Destination, dateTime)

    // ensure the new path doesn't already exist
    log.Printf("Mv(%s, %s)\n", newPath, file)
    if util.Exists(newPath) {
        util.MoveTo(sp.ErrorPath, file)
        log.Println("A file with that name already exists at the destination.")  // TODO This logging sucks.
        // TODO send an e-mail.
        return errors.New("A file with that name already exists at the destination.")
    }

    // move the file to its new home
    if err := util.Mv(newPath, file); err != nil {
        log.Println(err)
        return err
    }

    // add an entry to the hashmap db
    if err := sp.insertDb(newPath, hash); err != nil {
        return err  // TODO plain errors suck...
    }

    return nil
}

func (sp *Spool) insertDb(file, hash string) error {
    if _, exists := sp.db[hash]; !exists {
        sp.db[hash] = make([]string, 0, 5)
    }

    sp.db[hash] = append(sp.db[hash], file)

    return nil
}

/*
getHash calculates the md5 sum for a given filePath and returns the hex string
representation.

TODO Move to its own package?
*/
func getHash(filePath string) string {
    log.Println("Entering getHash(", filePath, ")")
    h := md5.New()

    inputFile, inputError := os.Open(filePath)
    if inputError != nil {
        log.Printf("An error occurred while opening the input file [%s].\n", filePath)
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
getDateTime reads the exif data from fname and returns a string representation
of the DateTimeOriginal tag.
*/
func getDateTime(fname string) (time.Time, error) {
    f, err := os.Open(fname)
    if err != nil {
        return time.Now(), err
    }

    x, err := exif.Decode(f)
    if err != nil {
        return time.Now(), err
    }

    date, err := x.Get(exif.DateTimeOriginal)
    if err != nil {
        var t time.Time
        return t, err
    }
    log.Println("Setting DateTimeOriginal to ", date.StringVal(), " on ", fname)
    t, err  := time.Parse("2006:01:02 15:04:05", date.StringVal())
    if err != nil {
        return time.Now(), err
    }

    return t, nil
}

/*
getDestination builds a full path where the origPath should be copied to based
on its DateTimeOriginal tag.
*/
func (sp *Spool) getDestination(origPath, newBasePath string, t time.Time) string {
    m   := int(t.Month())
    mon := strconv.Itoa(m)

    if m < 10 {
        mon = "0" + mon
    }

    dir   := filepath.Join(newBasePath, strconv.Itoa(t.Year()), mon)
    fname := t.Format("2006-01-02_15:04:05") + path.Ext(origPath)

    return filepath.Join(dir, fname)
}


