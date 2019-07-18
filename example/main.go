package main

import (
	"github.com/goroute/route"
	"log"
	"net/http"
)

type hello struct {
	Title string
}

func main() {
	mux := route.NewServeMux()

	mux.GET("/", func(c route.Context) error {
		return c.JSON(http.StatusOK, &hello{Title: "Hello, World!"})
	})

	log.Fatal(http.ListenAndServe(":9000", mux))
}
