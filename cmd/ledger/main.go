package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()

	server := &http.Server{
		Addr:              *addr,
		Handler:           newHandler("web"),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("个人记账本 http://%s", *addr)
	log.Fatal(server.ListenAndServe())
}

func newHandler(webDir string) http.Handler {
	api := http.NewServeMux()
	pending := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotImplemented)
		io.WriteString(w, `{"error":"接口待实现"}`)
	}
	api.HandleFunc("POST /api/records", pending)
	api.HandleFunc("GET /api/records/recent", pending)
	api.HandleFunc("GET /api/stats", pending)

	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	return mux
}
