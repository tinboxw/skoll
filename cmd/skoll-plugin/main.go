package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tinboxw/skoll/internal/handler/cli"
	"github.com/tinboxw/skoll/internal/plugin"
)

func main() {
	loader := plugin.NewFileLoader()
	manager := plugin.NewRuntimeManager(loader, plugin.NewTopologicalResolver())
	command := cli.NewPluginCommand(manager, loader, "")
	output, err := command.Handle(context.Background(), os.Args[1:])
	if output != "" {
		fmt.Println(output)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
