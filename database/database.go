package database

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/dgraph-io/badger/v3"
)

func InitDatabase() (*Database, error) {
	db := Database{}
	DB, err := NewBadgerDB("obsidiandb")
	db.DB = DB
	return &db, err
}

const (
	// Default BadgerDB discardRatio. It represents the discard ratio for the
	// BadgerDB GC.
	//
	// Ref: https://godoc.org/github.com/dgraph-io/badger#DB.RunValueLogGC
	badgerDiscardRatio = 0.5

	// Default BadgerDB GC interval
	badgerGCInterval = 10 * time.Minute
)

var (
	// BadgerAlertNamespace defines the alerts BadgerDB namespace.
	BadgerAlertNamespace = []byte("alerts")
)

// NewBadgerDB returns a new initialized BadgerDB database implementing the DB
// interface. If the database cannot be initialized, an error will be returned.
func NewBadgerDB(dataDir string) (DB, error) {
	if err := os.MkdirAll(dataDir, 0774); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(dataDir)
	opts.SyncWrites = true

	badgerDB, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	bdb := &BadgerDB{
		db: badgerDB,
		// logger: logger.With("module", "db"),
	}
	bdb.ctx, bdb.cancelFunc = context.WithCancel(context.Background())

	go bdb.runGC()

	entropy := rand.New(rand.NewSource(time.Unix(1000000, 0).UnixNano()))
	// sub-ms safe ULID generator
	ulidSource := NewMonotonicULIDsource(entropy)
	bdb.ulidSource = ulidSource
	return bdb, nil
}

// Get implements the DB interface. It attempts to get a value for a given key
// and namespace. If the key does not exist in the provided namespace, an error
// is returned, otherwise the retrieved value.
func (bdb *BadgerDB) Get(namespace, key []byte) (value []byte, err error) {
	err = bdb.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(badgerNamespaceKey(namespace, key))
		if err != nil {
			return err
		}

		value = make([]byte, item.ValueSize())

		value, err = item.ValueCopy(value)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return value, nil
}

// Set implements the DB interface. It attempts to store a value for a given key
// and namespace. If the key/value pair cannot be saved, an error is returned.
func (bdb *BadgerDB) Set(namespace []byte, objs []Object) error {
	batch := bdb.db.NewWriteBatch()
	for _, obj := range objs {
		err := batch.Set(badgerNamespaceKey(namespace, obj.Key), obj.Value)
		if err != nil {
			log.Printf("failed to set key %s for namespace %s: %v\n", obj, namespace, err)
			return err
		}
	}
	return batch.Flush()
}

func (bdb *BadgerDB) Delete(namespace, key []byte) error {
	err := bdb.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(badgerNamespaceKey(namespace, key))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (bdb *BadgerDB) DeleteNamespace(namespace []byte) error {
	batch := bdb.db.NewWriteBatch()
	err := bdb.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = namespace
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()

			if err := batch.Delete(k); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return batch.Flush()
}

func (bdb *BadgerDB) ReadIteratorCopy(prefix []byte, reverse bool, action func(k []byte, v []byte) (bool, error)) error {

	err := bdb.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.Prefix = prefix
		opts.Reverse = reverse
		it := txn.NewIterator(opts)
		defer it.Close()
		var willStop bool
		if !reverse {
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				v, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				willStop, err = action(k, v)
				if err != nil {
					return err
				}
				if willStop {
					return nil
				}
			}
		} else {
			prefix := append(prefix, 0xFF)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				k := item.Key()
				v, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				willStop, err = action(k, v)
				if err != nil {
					return err
				}
				if willStop {
					return nil
				}
			}
		}
		return nil
	})
	return err
}

func (bdb *BadgerDB) ReadIteratorNonCopy(prefix []byte, reverse bool, action func(k []byte, v []byte) (bool, error)) error {
	err := bdb.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.Prefix = prefix
		opts.Reverse = reverse
		it := txn.NewIterator(opts)
		var willStop bool
		defer it.Close()
		if !reverse {
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				err := item.Value(func(v []byte) error {
					var err error
					willStop, err = action(k, v)
					if err != nil {
						log.Fatal(err)
					}
					return nil
				})
				if err != nil {
					return err
				}
				if willStop {
					return nil
				}
			}
		} else {
			prefix := append(prefix, 0xFF)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				k := item.Key()
				err := item.Value(func(v []byte) error {
					var err error
					willStop, err = action(k, v)
					if err != nil {
						log.Fatal(err)
					}
					return nil
				})
				if err != nil {
					return err
				}
				if willStop {
					return nil
				}
			}
		}

		return nil
	})
	return err
}

// Has implements the DB interface. It returns a boolean reflecting if the
// datbase has a given key for a namespace or not. An error is only returned if
// an error to Get would be returned that is not of type badger.ErrKeyNotFound.
func (bdb *BadgerDB) Has(namespace, key []byte) (ok bool, err error) {
	_, err = bdb.Get(namespace, key)
	switch err {
	case badger.ErrKeyNotFound:
		ok, err = false, nil
	case nil:
		ok, err = true, nil
	}

	return
}

// Close implements the DB interface. It closes the connection to the underlying
// BadgerDB database as well as invoking the context's cancel function.
func (bdb *BadgerDB) Close() error {
	bdb.cancelFunc()
	return bdb.db.Close()
}

// runGC triggers the garbage collection for the BadgerDB backend database. It
// should be run in a goroutine.
func (bdb *BadgerDB) runGC() {
	ticker := time.NewTicker(badgerGCInterval)
	for {
		select {
		case <-ticker.C:
			err := bdb.db.RunValueLogGC(badgerDiscardRatio)
			if err != nil {
				// don't report error when GC didn't result in any cleanup
				if err == badger.ErrNoRewrite {
					log.Printf("no BadgerDB GC occurred: %v\n", err)
				} else {
					log.Printf("failed to GC BadgerDB: %v\n", err)
				}
			}

		case <-bdb.ctx.Done():
			return
		}
	}
}

func (bdb *BadgerDB) CreateULID(t time.Time) ([]byte, error) {
	id, _ := bdb.ulidSource.New(t)
	return id.MarshalBinary()
}

// badgerNamespaceKey returns a composite key used for lookup and storage for a
// given namespace and key.
func badgerNamespaceKey(namespace, key []byte) []byte {
	prefix := []byte(fmt.Sprintf("%s", namespace))
	return append(prefix, key...)
}
