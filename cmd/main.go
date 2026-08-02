package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	dohttpstd "github.com/samber/do/http/std/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/di"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	injector := di.New()

	mux := http.NewServeMux()
	mux.Handle("/debug/do/", dohttpstd.Use("/debug/do", injector))
	go func() {
		fmt.Println("Server Run: " + "localhost:8080/debug/do")
		srv := &http.Server{
			Addr:              ":8080",
			Handler:           mux,
			ReadHeaderTimeout: 3 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()
	debug := do.ExplainInjector(injector)
	println(debug.String())

	return di.Run(injector)
}
