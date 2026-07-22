package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginclient"
)

func main() {
	client, err := pluginclient.FromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	host, err := client.HostServices()
	if err != nil {
		log.Fatal(err)
	}
	address := strings.TrimSpace(os.Getenv("SKOLL_PLUGIN_ADDRESS"))
	if address == "" {
		log.Fatal(errors.New("SKOLL_PLUGIN_ADDRESS is required"))
	}
	handler, err := newHandler(host)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("%s listening on %s", pluginID, address)
	log.Fatal(server.ListenAndServe())
}
