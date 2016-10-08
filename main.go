package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/jhunt/go-tilesetter/tile"
	"github.com/jhunt/go-tilesetter/web"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "USAGE: %s COMMAND [arguments...]\n", os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "inspect":
		inspect()

	case "api":
		api()

	default:
		fmt.Fprintf(os.Stderr, "Unrecognized sub-command '%s'\n", os.Args[1])
		os.Exit(1)
	}
}

func inspect() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "USAGE: %s inspect /path/to/tile.pivotal\n", os.Args[0])
		os.Exit(1)
	}

	fmt.Printf("# %s\n", os.Args[2])
	t, err := tile.Unpack(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	s, err := yaml.Marshal(t)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", s)
	return
}

func api() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Fprintf(os.Stderr, "USAGE: %s api /path/to/ui/assets [host:port]\n", os.Args[0])
		os.Exit(1)
	}

	bind := ":5001"
	if len(os.Args) == 4 {
		bind = os.Args[3]
	}

	fmt.Printf("starting up tilesetter web ui on %s\n", bind)
	fmt.Printf("(sourcing assets from %s)\n", os.Args[2])
	web.Listen(os.Args[2], bind)
}
