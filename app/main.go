package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", "8080"),
		Handler: nil,
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}
