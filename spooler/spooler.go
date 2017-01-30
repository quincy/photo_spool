package spooler

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/quincy/goutil/file"
	"github.com/quincy/photo_spool/db"
	"github.com/rwcarlsen/goexif/exif"
)

type Spooler struct {
	Destination string
	dbPath      string
	ErrorPath   string
	database    db.Db
	noop        bool
	closed      bool
}

var JPEGPattern string = `\.jpe?g$`
var PNGPattern string = `\.png$`

var ImagePattern string = `(?i:` + strings.Join([]string{JPEGPattern, PNGPattern}, "|") + `$)`

// New creates and returns a new *Spooler.
func New(dbPath, destination, errorPath string, noop bool) (*Spooler, error) {
	sp := new(Spooler)
	sp.noop = noop

	if !file.Exists(destination) {
		if _, err := os.Create(destination); err != nil {
			return nil, errors.Wrap(err, "could not create file "+destination)
		}
	}
	sp.Destination = destination

	if !file.Exists(errorPath) {
		if _, err := os.Create(errorPath); err != nil {
			return nil, errors.Wrap(err, "could not create file "+errorPath)
		}
	}
	sp.ErrorPath = errorPath

	if sp.noop {
		sp.database = db.NewNoopDb()
	} else {
		database, err := db.NewMapFileDb(dbPath)
		if err != nil {
			return nil, errors.Wrap(err, "could not create a new MapFileDb at "+dbPath)
		}
		sp.database = database
	}

	sp.closed = false
	return sp, nil
}

// Close closes the spooler, in turn closing the database.
func (sp *Spooler) Close() error {
	if sp.closed {
		return fmt.Errorf("This Spooler is already closed.")
	}

	err := sp.database.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close database "+sp.dbPath)
	}

	sp.closed = true
	return nil
}

// Spool copies the photo given by filename to the correct directory with the correct name.
func (sp *Spooler) Spool(filename string) error {
	if sp.closed {
		return fmt.Errorf("This Spooler is already closed.")
	}

	// calculate an md5 sum for the file
	hash, err := getHash(filename)
	if err != nil {
		return errors.Wrap(err, "error calculating hash for "+filename)
	}

	// get the Time from the DateTimeOriginal exif tag
	dateTime, err := getDateTime(filename)
	if err != nil {
		log.Printf("Could not read the DateTimeOriginal tag. %v", err)
		log.Printf("Moving %s to %s.\n", filename, sp.ErrorPath)
		if sp.noop {
			log.Println("DRY RUN Skipping move file.")
		} else if mverr := file.MoveTo(sp.ErrorPath, filename); mverr != nil {
			log.Fatal(mverr)
		}
		return errors.Wrap(err, "failed to lookup DateTimeOriginal tag from "+filename)
	}

	// check if the hash already exists in the db
	if sp.database.ContainsKey(hash) {
		msg := "A db entry already exists for " + filename + "."
		errorName := filepath.Join(sp.ErrorPath, strings.Join([]string{filepath.Base(filename), "DUPLICATE"}, "."))
		log.Printf("Mv(%s, %s)\n", errorName, filename)
		if sp.noop {
			log.Println("DRY RUN Skipping move file.")
		} else {
			err := file.Mv(errorName, filename)
			if err != nil {
				return errors.Wrap(err, "error moving "+filename+" to "+errorName)
			}
		}
		return errors.New(msg)
	}

	// calculate the path to copy the file to, including a new file name
	newPath := sp.getDestination(filename, sp.Destination, dateTime)

	// ensure the new path doesn't already exist
	for file.Exists(newPath) {
		dateTime = dateTime.Add(1 * time.Second)
		newPath = sp.getDestination(filename, sp.Destination, dateTime)
	}

	log.Printf("Mv(%s, %s)\n", newPath, filename)
	if file.Exists(newPath) {
		fields := strings.Split(filepath.Base(filename), ".")
		errorName := filepath.Join(sp.ErrorPath, strings.Join([]string{fields[0], filepath.Base(newPath)}, "::"))
		if sp.noop {
			log.Println("DRY RUN Skipping move file.")
		} else {
			err := file.Mv(errorName, filename)
			if err != nil {
				return errors.Wrap(err, "error moving "+filename+" to "+errorName)
			}
		}
		msg := "A file with that named " + newPath + " already exists at the destination.  Moving to " + errorName
		// TODO send an e-mail.
		return errors.New(msg)
	}

	// move the file to its new home
	if sp.noop {
		log.Println("DRY RUN Skipping move file.")
	} else if err := file.Mv(newPath, filename); err != nil {
		return errors.Wrap(err, "error moving "+filename+" to "+newPath)
	}

	// add an entry to the hashmap db
	if err := sp.database.Insert(newPath, hash); err != nil {
		return errors.Wrap(err, "could not insert "+newPath+" with hash "+hash+" into database.")
	}

	return nil
}

// getHash calculates the md5 sum for a given filePath and returns the hex string
// representation.
// TODO move to its own package?
func getHash(filePath string) (string, error) {
	log.Println("Entering getHash(", filePath, ")")
	h := md5.New()

	inputFile, inputError := os.Open(filePath)
	if inputError != nil {
		return "", errors.Wrap(inputError, "could not open input file "+filePath)
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
	return sum, nil
}

// getDateTime reads the exif data from fname and returns a string representation
// of the DateTimeOriginal tag.
// TODO move to its own package?
func getDateTime(fname string) (time.Time, error) {
	f, err := os.Open(fname)
	if err != nil {
		return time.Now(), errors.Wrap(err, "could not open file "+fname)
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return time.Now(), errors.Wrap(err, "cound not decode exif data in file "+fname)
	}

	date, err := x.Get(exif.DateTimeOriginal)
	if err != nil {
		return time.Now(), errors.Wrap(err, "could not extract exif tag DateTimeOriginal from "+fname)
	}
	dateStr, err := date.StringVal()
	if err != nil {
		return time.Now(), nil
	}
	log.Println("Setting DateTimeOriginal to ", dateStr, " on ", fname)
	t, err := time.Parse("2006:01:02 15:04:05", dateStr)
	if err != nil {
		return time.Now(), errors.Wrap(err, "could not parse DateTimeOriginal value "+dateStr)
	}

	return t, nil
}

// getDestination builds a full path where the origPath should be copied to based on the newBasePath and the given time t.
// If t is "2006-01-02_150405" then this function will append "/2006/01/2006-01-02_150405" onto newBasePath and return
// the resulting path.
func (sp *Spooler) getDestination(origPath, newBasePath string, t time.Time) string {
	m := int(t.Month())
	mon := strconv.Itoa(m)

	if m < 10 {
		mon = "0" + mon
	}

	suffix := strings.ToLower(path.Ext(origPath))
	if suffix == "jpeg" {
		suffix = "jpg"
	}

	dir := filepath.Join(newBasePath, strconv.Itoa(t.Year()), mon)
	fname := t.Format("2006-01-02_150405") + suffix
	return filepath.Join(dir, fname)
}
