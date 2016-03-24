package db

type Db interface {
	Load() error
	Write() error
	Close() error
	Insert(file, hash string) error
	ContainsKey(string) bool
}
