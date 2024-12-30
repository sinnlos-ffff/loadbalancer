package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
    port := os.Args[1]

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello from server on port %s\n", port)
    })

    log.Printf("Server starting on port %s\n", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}
