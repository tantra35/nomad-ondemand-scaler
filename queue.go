package main

import (
	"sync"
)

type QueueItem[T any] struct {
	item T
	id   string
}

type Queue[T any] struct {
	lock sync.Mutex

	itemsIds    map[string]int
	items       []*QueueItem[T]
	intemsCh    chan *QueueItem[T]
	waiyitemsCh chan struct{}
}

func NewQueue[T any]() *Queue[T] {
	q := &Queue[T]{
		itemsIds:    make(map[string]int),
		intemsCh:    make(chan *QueueItem[T]),
		waiyitemsCh: make(chan struct{}, 1),
	}

	go func() {
		var item *QueueItem[T]

		for {
			q.lock.Lock()
			if len(q.items) > 0 {
				item = q.items[0]
			}
			q.lock.Unlock()

			if item != nil {
				q.intemsCh <- item
				item = nil
			} else {
				<-q.waiyitemsCh
			}
		}
	}()

	return q
}

func (q *Queue[T]) Enqueue(item T, itemId string) {
	q.lock.Lock()
	defer q.lock.Unlock()

	index := len(q.items)
	q.items = append(q.items, &QueueItem[T]{item, itemId})
	q.itemsIds[itemId] = index

	select {
	case q.waiyitemsCh <- struct{}{}:
	default:
	}
}

func (q *Queue[T]) Dequeue() T {
	var item T
	var lquieueitem *QueueItem[T]

	for {
		<-q.intemsCh

		q.lock.Lock()
		if len(q.items) > 0 {
			lquieueitem, q.items = q.items[0], q.items[1:]
			for id, lindex := range q.itemsIds {
				if lindex > 0 {
					q.itemsIds[id] -= 1
				}
			}

			if q.itemsIds[lquieueitem.id] == -1 {
				delete(q.itemsIds, lquieueitem.id)
				q.lock.Unlock()
				continue
			}

			delete(q.itemsIds, lquieueitem.id)
		}
		q.lock.Unlock()

		if lquieueitem != nil {
			item = lquieueitem.item
			break
		}
	}

	return item
}

func (q *Queue[T]) Remove(id string) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	index := -1
	if itemIndex, lok := q.itemsIds[id]; lok {
		index = itemIndex
		q.itemsIds[id] = -1
	}

	if index == -1 {
		return false
	}

	q.items = append(q.items[:index], q.items[index+1:]...)

	for id, lindex := range q.itemsIds {
		if lindex > index {
			q.itemsIds[id] -= 1
		}
	}

	return true
}
