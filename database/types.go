package database

import (
	"context"
	"time"

	"github.com/dgraph-io/badger/v3"
)

type Database struct {
	DB DB
}

type Object struct {
	Key   []byte
	Value []byte
}
type (
	// DB defines an embedded key/value store database interface.
	DB interface {
		Get(namespace, key []byte) (value []byte, err error)
		Set(namespace []byte, objs []Object) error
		Delete(namespace, key []byte) error
		DeleteNamespace(namespace []byte) error
		ReadIteratorCopy(prefix []byte, reverse bool, action func(k []byte, v []byte) (bool, error)) error
		ReadIteratorNonCopy(prefix []byte, reverse bool, action func(k []byte, v []byte) (willStop bool, err error)) error
		Has(namespace, key []byte) (bool, error)
		Close() error

		CreateULID(t time.Time) ([]byte, error)
	}

	// BadgerDB is a wrapper around a BadgerDB backend database that implements
	// the DB interface.
	BadgerDB struct {
		db         *badger.DB
		ctx        context.Context
		cancelFunc context.CancelFunc
		ulidSource *MonotonicULIDsource
	}
)
