package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/quincy/goutil/file"
)

// MapFileDb is an implementation of Db which uses a file with a serialized map to store the data.
type MapFileDb struct {
	FilePath string
	store    map[string][]string
	closed   bool
}

// NewMapFileDb creates a new MapFileDb and returns its address.
func NewMapFileDb(path string) (Db, error) {
	m, err := load(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load db from file "+path)
	}
	d := &MapFileDb{
		FilePath: path,
		store:    m,
		closed:   false}
	return d, nil
}

// load deserializes the md5 json database from the given filePath.
func load(path string) (map[string][]string, error) {
	var db map[string][]string

	if file.Exists(path) {
		json_bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return db, errors.Wrap(err, "could not read file "+path)
		}

		err = json.Unmarshal(json_bytes, &db)
		if err != nil {
			return db, errors.Wrap(err, "failed to unmarhsal json ["+string(json_bytes)+"]")
		}
	} else {
		db = make(map[string][]string)
	}

	return db, nil
}

// Load reads in the database from disk and stores it in this Db.
func (d *MapFileDb) Load() error {
	if d.closed {
		return fmt.Errorf("Database has already been closed.")
	}

	m, err := load(d.FilePath)
	if err != nil {
		return err
	}

	d.store = m
	return nil
}

// Write serializes the md5 database to json and writes it to disk.
func (d *MapFileDb) Write() error {
	if d.closed {
		return fmt.Errorf("Database has already been closed.")
	}

	b, err := json.Marshal(d.store)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(d.FilePath, b, 0644)
	return err
}

// Close writes any unsaved changes back to disk and closes the Db so it can no longer be used.
func (d *MapFileDb) Close() error {
	if d.closed {
		return fmt.Errorf("Database has already been closed.")
	}

	err := d.Write()
	if err != nil {
		return err
	}
	d.closed = true
	return nil
}

// Insert a key/value pair into the database.
func (d *MapFileDb) Insert(file, hash string) error {
	if d.closed {
		return fmt.Errorf("Database has already been closed.")
	}

	if _, exists := d.store[hash]; !exists {
		d.store[hash] = make([]string, 0, 5)
	}

	d.store[hash] = append(d.store[hash], file)

	return nil
}

// ContainsKey returns true if the database contains the given key, and false otherwise.
func (d *MapFileDb) ContainsKey(key string) bool {
	if _, ok := d.store[key]; ok {
		return true
	}

	return false
}
