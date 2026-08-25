package main

import (
	"fmt"
	"golang.org/x/sync/errgroup"
)

func main() {
	var eg errgroup.Group
	eg.Go(func() error {
		fmt.Println("Hello")
        return nil
	})
	_ = eg.Wait()
}
