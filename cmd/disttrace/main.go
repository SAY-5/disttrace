// disttrace serve — runs the trace ingest + analysis HTTP server.

package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/SAY-5/disttrace/api"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	flag.Parse()
	srv := api.New()
	mux := http.NewServeMux()
	srv.Routes(mux)
	log.Printf("disttrace listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
