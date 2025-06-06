//go:build !js
// +build !js

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Starting server on http://localhost:8080")
	// Serve files from the current directory, which is expected to be 'web'
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}