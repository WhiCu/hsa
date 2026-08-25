package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Go(func() {
		fmt.Println("Hello")
	})
	wg.Wait()
}
