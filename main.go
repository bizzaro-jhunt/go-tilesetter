package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/jhunt/go-tilesetter/tile"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s path/to/tile.pivotal\n", os.Args[0])
		os.Exit(1)
	}

	fmt.Printf("# %s\n", os.Args[1])
	t, err := tile.Unpack(os.Args[1])
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
