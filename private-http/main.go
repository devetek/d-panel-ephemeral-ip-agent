package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
)

type PageData struct {
	Title   string
	Content string
}

func main() {
	// Define a handler function for the "/hello" route
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("private-http/templates/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := PageData{
			Title:   "My Dynamic Page",
			Content: "This content is dynamically generated!",
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Start the HTTP server on port 3000
	fmt.Println("Server starting on port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
