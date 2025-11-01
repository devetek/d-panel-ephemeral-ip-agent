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

		log.Println("X-Forwarded-For:", r.Header.Get("X-Forwarded-For"))
		log.Println("User-Agent:", r.Header.Get("User-Agent"))
		log.Println("Access:", r.URL.Path)

		data := PageData{
			Title:   "Mau online ken PC di rumah kalian ?",
			Content: "Pake dPanel makanya, jangan lupa daftar di <a href=\"https://cloud.terpusat.com/\">disini</a>.",
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
