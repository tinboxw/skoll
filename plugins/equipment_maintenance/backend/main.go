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
	store, err := OpenStore(os.Getenv("SKOLL_PLUGIN_DATA_DIR"))
	if err != nil {
		log.Fatal(err)
	}
	server, err := NewServer(store, Host{
		Transactions: host.Transactions, DataScopes: host.DataScopes, Files: host.Files, Audit: host.Audit,
		Config: host.Config, Workflows: host.Workflows, Jobs: host.Jobs,
	})
	if err != nil {
		log.Fatal(err)
	}
	address := strings.TrimSpace(os.Getenv("SKOLL_PLUGIN_ADDRESS"))
	if address == "" {
		log.Fatal(errors.New("SKOLL_PLUGIN_ADDRESS is required"))
	}
	httpServer := &http.Server{Addr: address, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("%s listening on %s", pluginID, address)
	log.Fatal(httpServer.ListenAndServe())
}
