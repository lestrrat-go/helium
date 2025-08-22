package orderedmap

import (
	"errors"
	"iter"
)

var ErrDuplicateEntry = errors.New("duplicate entry")

type Map[K comparable, V any] struct {
	entries []K
	keys    map[K]V
}

func New[K comparable, V any]() *Map[K, V] {
	// TODO: use pooling
	return &Map[K, V]{
		entries: make([]K, 0),
		keys:    make(map[K]V),
	}
}

func (m *Map[K, V]) Set(key K, value V) error {
	_, exists := m.keys[key]
	if exists {
		return ErrDuplicateEntry
	}
	m.entries = append(m.entries, key)
	m.keys[key] = value
	return nil
}

func (m *Map[K, V]) Len() int {
	return len(m.entries)
}

func (m *Map[K, V]) Range() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range m.entries {
			v := m.keys[k]
			if !yield(k, v) {
				break
			}
		}
	}
}
