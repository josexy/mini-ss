package ordmap

import (
	"container/list"
	"sync"
)

type entry struct {
	key   any
	value any
}

type OrderedMap struct {
	mu     sync.RWMutex
	once   sync.Once
	record map[any]*list.Element
	ll     *list.List
}

func (m *OrderedMap) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ll == nil {
		return 0
	}
	return m.ll.Len()
}

func (m *OrderedMap) Store(key, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.once.Do(func() {
		m.record = make(map[any]*list.Element, 16)
		m.ll = list.New()
	})

	e := &entry{key: key, value: value}
	insE := m.ll.PushBack(e)
	m.record[key] = insE
}

func (m *OrderedMap) Load(key any) (value any, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_v, _ok := m.record[key]
	if !_ok {
		return nil, _ok
	}
	return _v.Value.(*entry).value, _ok
}

func (m *OrderedMap) Delete(key any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_v, _ok := m.record[key]
	if !_ok {
		return
	}
	m.ll.Remove(_v)
	delete(m.record, key)
}

func (m *OrderedMap) First() (val any, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ll == nil {
		return
	}
	f := m.ll.Front()
	if f == nil {
		ok = false
		return
	}
	return f.Value.(*entry).value, true
}

func (m *OrderedMap) Range(f func(key, value any) bool) {
	if m.ll == nil {
		return
	}
	for x := m.ll.Front(); x != nil; x = x.Next() {
		v := x.Value.(*entry)
		if !f(v.key, v.value) {
			break
		}
	}
}

func (m *OrderedMap) UnorderedRange(f func(key, value any) bool) {
	for k, v := range m.record {
		if !f(k, v.Value.(*entry).value) {
			break
		}
	}
}
