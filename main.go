//nolint:forbidigo
package main

import (
	"fmt"
	"log"
	"os"

	"golift.io/turbovanityurls/pkg/service"
)

// Version is injected at build time.
var Version = "development" //nolint:gochecknoglobals

func main() {
	flags := service.ParseFlags(os.Args[1:])
	if flags.ShowVer {
		fmt.Printf("turbovanityurls v%v\n", Version)
		os.Exit(0)
	}

	server, err := service.Setup(flags)
	if err != nil {
		log.Fatal(err)
	}

	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}
}
