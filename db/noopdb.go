package db

import (
	"log"
)

type NoopDb struct {
}

func NewNoopDb() *NoopDb {
	return &NoopDb{}
}

func (d *NoopDb) Load() error {
	log.Println("DRY RUN Skipping load database.")
	return nil
}

func (d *NoopDb) Write() error {
	log.Println("DRY RUN Skipping write database.")
	return nil
}

func (d *NoopDb) Insert(file, hash string) error {
	return nil
}

func (d *NoopDb) Close() error {
	return nil
}

func (d *NoopDb) ContainsKey(key string) bool {
	return false
}
