package store

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"google.golang.org/protobuf/proto"

	snsv1 "github.com/lodek/sns/gen/sns/v1"
)

var (
	oneShotPrefix   = []byte("oneshot/")
	recurringPrefix = []byte("recurring/")
)

type Store struct {
	db *badger.DB
}

func New(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir).WithLoggingLevel(badger.WARNING)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) PutOneShotAlert(_ context.Context, a *snsv1.OneShotAlert) error {
	val, err := proto.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal oneshot alert: %w", err)
	}
	key := append(oneShotPrefix, []byte(a.Id)...)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

func (s *Store) PutRecurringAlert(_ context.Context, a *snsv1.RecurringAlert) error {
	val, err := proto.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal recurring alert: %w", err)
	}
	key := append(recurringPrefix, []byte(a.Id)...)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

func (s *Store) ListOneShotAlerts(_ context.Context) ([]*snsv1.OneShotAlert, error) {
	var alerts []*snsv1.OneShotAlert
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(oneShotPrefix); it.ValidForPrefix(oneShotPrefix); it.Next() {
			item := it.Item()
			if err := item.Value(func(val []byte) error {
				a := &snsv1.OneShotAlert{}
				if err := proto.Unmarshal(val, a); err != nil {
					return err
				}
				alerts = append(alerts, a)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return alerts, err
}

func (s *Store) ListRecurringAlerts(_ context.Context) ([]*snsv1.RecurringAlert, error) {
	var alerts []*snsv1.RecurringAlert
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(recurringPrefix); it.ValidForPrefix(recurringPrefix); it.Next() {
			item := it.Item()
			if err := item.Value(func(val []byte) error {
				a := &snsv1.RecurringAlert{}
				if err := proto.Unmarshal(val, a); err != nil {
					return err
				}
				alerts = append(alerts, a)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return alerts, err
}

func (s *Store) DeleteOneShotAlert(_ context.Context, id string) error {
	key := append(oneShotPrefix, []byte(id)...)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *Store) DeleteRecurringAlert(_ context.Context, id string) error {
	key := append(recurringPrefix, []byte(id)...)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// DeleteAlert tries to delete an alert by ID from both prefixes.
func (s *Store) DeleteAlert(_ context.Context, id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		oneShotKey := append(oneShotPrefix, []byte(id)...)
		recurringKey := append(recurringPrefix, []byte(id)...)
		err1 := txn.Delete(oneShotKey)
		err2 := txn.Delete(recurringKey)
		// Badger Delete doesn't error on missing keys, so both are safe.
		if err1 != nil {
			return err1
		}
		return err2
	})
}
