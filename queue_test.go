package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	queue := NewQueue[int]()
	queue.Enqueue(1, "a")
	queue.Enqueue(2, "b")
	queue.Enqueue(3, "c")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 2; i++ {
			time.Sleep(5 * time.Second)
			item := queue.Dequeue()

			fmt.Println(item)
		}
	}()

	if queue.Remove("a") {
		fmt.Println("remove item a from queue")
	}

	time.Sleep(3 * time.Second)
	if queue.Remove("c") {
		fmt.Println("remove item a from queue")
	}

	queue.Enqueue(5, "d")

	wg.Wait()
}
