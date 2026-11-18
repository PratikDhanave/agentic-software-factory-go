package main

import (
	"fmt"
	"sync"
)

// BadExample has multiple code issues
type BadExample struct {
	mu sync.Mutex
	data map[string]string
}

// This function has error not handled properly
func (b *BadExample) ProcessData(key string) error {
	b.mu.Lock() // TODO: need to add defer unlock
	val := b.data[key]
	fmt.Println(val)

	// Missing error handling
	_ = performOperation(val)
	return nil
}

// Large function that should be broken down
func (b *BadExample) LargeFunction() {
	// Line 1
	fmt.Println("line 1")
	// Line 2
	fmt.Println("line 2")
	// Line 3
	fmt.Println("line 3")
	// ... continues for 200+ lines
	for i := 0; i < 50; i++ {
		fmt.Println(i)
		fmt.Println(i)
		fmt.Println(i)
		fmt.Println(i)
		fmt.Println(i)
	}
	// FIXME: Refactor this function
}

// Missing error handling
func performOperation(data string) string {
	if data == "" {
		return "empty"
	}
	return data
}

// No tests defined for this package
