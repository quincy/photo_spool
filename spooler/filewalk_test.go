package spooler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"github.com/pkg/errors"
)

func TestGetWalkFunc(t *testing.T) {
	noop := false
	rootPath := "testDir"
	spoolPath := "testSpool"
	dbPath := "testDB"
	destinationPath := "testDestination"
	errorPath := "testError"
	emptyDir := filepath.Join(spoolPath, "emptyDir")

	paths := []string{
		spoolPath,
		emptyDir,
		dbPath,
		destinationPath,
		errorPath,
	}

	err := setup(rootPath, paths)
	if err != nil {
		t.Errorf("%+v", err)
	}

	fakeSpool, err := New(dbPath, destinationPath, errorPath, noop)
	if err != nil {
		t.Errorf("%+v", err)
	}

	walkFunc := GetWalkFunc(fakeSpool, spoolPath, noop)
	err = filepath.Walk(spoolPath, walkFunc)

	_, err = os.Open(emptyDir)
	if err != nil && !os.IsNotExist(err) {
		t.Errorf("%+v", err)
	}

	err = teardown(rootPath, paths)
	if err != nil {
		t.Errorf("%+v", err)
	}
}

func setup(root string, paths []string) error {
	err := createDir(root)
	if err != nil {
		return errors.Wrap(err, "error creating root dir")
	}

	err = os.Chdir(root)
	if err != nil {
		return errors.Wrap(err, "failed to change directory to " + root)
	}

	for _, path := range paths {
		fmt.Fprintf(os.Stderr, "creating path %s\n", path)
		err = createDir(path)
		if err != nil && err != os.ErrExist {
			return errors.Wrap(err, "failed to create dir")
		}
	}

	return nil
}

func teardown(root string, paths []string) error {
	err := os.Chdir(root)
	if err != nil {
		return err
	}

	reversed := reverse(paths)

	for _, path := range reversed {
		fmt.Fprintf(os.Stderr, "removing path %s\n", path)
		err = os.RemoveAll(path)
		if err != nil {
			return err
		}
	}

	err = os.Chdir("..")
	if err != nil {
		return err
	}

	err = os.RemoveAll(root)
	if err != nil {
		return err
	}

	return nil
}

func reverse(orig []string) []string {
	reversed := make([]string, 0)
	for i := len(orig) - 1; i <= 0; i-- {
		reversed = append(reversed, orig[i])
	}
	return reversed
}

func createFile(path string) error {
	// detect if file exists
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			return errors.Wrap(err, "could not create file " + path)
		}
		defer file.Close()
	}

	return nil
}

func createDir(path string) error {
	// detect if file exists
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		err = os.Mkdir(path, os.ModeDir)
		if err != nil {
			return errors.Wrap(err, "could not create file " + path)
		}
	}

	return nil
}
