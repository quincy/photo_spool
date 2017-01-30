package spooler

import (
	"github.com/quincy/goutil/file"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// GetWalkFunc returns a filepath.WalkFunc which looks for files matching image patterns and passes them to the *Spooler
// spool.  Any empty directories which are encountered are pruned in the process.
func GetWalkFunc(spool *Spooler, spoolPath string, noop bool) func(string, os.FileInfo, error) error {
	return func(filePath string, f os.FileInfo, err error) error {
		log.Println("Visiting", filePath)

		// If we visit a directory check to see if it's empty and prune if so.
		if f.IsDir() {
			return pruneDir(filepath.Dir(filePath), spoolPath, noop)
		}

		matched, err := regexp.MatchString(ImagePattern, filePath)
		if err != nil {
			log.Fatalf("Error compiling regular expression '%s'.  %v", ImagePattern, err)
		}

		if !matched {
			log.Printf("Found unhandled file type [%s].  Moving the file to %s.\n", filePath, spool.ErrorPath)
			if noop {
				log.Println("DRY RUN Skipping move file.")
			} else {
				file.MoveTo(spool.ErrorPath, filePath)
			}
			return nil
		}

		spoolError := spool.Spool(filePath)
		if spoolError != nil {
			log.Println(spoolError)
		}

		return pruneDir(filepath.Dir(filePath), spoolPath, noop)
	}
}

// pruneDir removes the directory d if it is empty.  Otherwise prune does nothing.
func pruneDir(d, spoolPath string, noop bool) error {
	if file.DirIsEmpty(d) && d != spoolPath {
		log.Printf("Pruning empty directory [%s].\n", d)
		if noop {
			log.Println("DRY RUN Skipping delete directory.")
		} else {
			err := os.Remove(d)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
