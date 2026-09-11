package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	if err := http.ListenAndServe(":"+os.Args[1], nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
