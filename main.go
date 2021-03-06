package main

import (
	"fmt"
	_ "html/template"
	"log"
	"net/http"
	"runtime"
)

func dummyHandler(w http.ResponseWriter, r *http.Request, c *Config) {
	fmt.Fprintf(w, "HANDLING")
}

type ConfigHandler func(w http.ResponseWriter, r *http.Request, c *Config)

// Make a handler function including the given config
func makeHandler(h ConfigHandler, c *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r, c)
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/list", http.StatusTemporaryRedirect)
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	// Load config, sources, articles... from disk
	config := loadConfig()

	// Start the update loop: it will regularly update the sources
	go updateLoop(config)

	// Visible index

	http.HandleFunc("/", mainHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/list", makeHandler(handleList, config))
	http.HandleFunc("/view/", makeHandler(handleView, config))
	http.HandleFunc("/edit/", makeHandler(handleEdit, config))
	http.HandleFunc("/options", makeHandler(handleOptions, config))

	// Machine-only API (via AJAX)
	http.HandleFunc("/api/source/list", makeHandler(dummyHandler, config))
	http.HandleFunc("/api/source/add", makeHandler(handleApiAdd, config))
	http.HandleFunc("/api/source/remove/", makeHandler(handleApiRemove, config))
	http.HandleFunc("/api/source/edit/", makeHandler(handleApiEdit, config))
	http.HandleFunc("/api/source/refresh/", makeHandler(handleApiRefresh, config))
	http.HandleFunc("/api/article/list/", makeHandler(handleApiListArticles, config))
	http.HandleFunc("/api/article/get/", makeHandler(handleApiGetArticle, config))
	http.HandleFunc("/api/article/read/", makeHandler(handleApiReadArticle, config))
	http.HandleFunc("/api/options", makeHandler(handleApiOptions, config))

	log.Println("Listening to port " + config.Port + "...")
	log.Fatal(http.ListenAndServe("localhost:"+config.Port, nil))
	log.Println("Exiting.")
}
