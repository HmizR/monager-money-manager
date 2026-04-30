package main

import (
	"log"
	"net/http"
	"os"
)

func serve(path, file string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", 405)
			return
		}

		http.ServeFile(w, r, file)
	}
}

func main() {
	log.SetOutput(os.Stdout)

	http.HandleFunc("/", serve("/", "index.html"))
	http.HandleFunc("/app.js", serve("/app.js", "app.js"))
	http.HandleFunc("/style.css", serve("/style.css", "style.css"))
	http.HandleFunc("/config.js", serve("/config.js", "config.js"))

	log.Println("Serving on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}