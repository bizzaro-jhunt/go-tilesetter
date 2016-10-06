package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	//"gopkg.in/yaml.v2"
	"github.com/goware/prefixer"
)

var tgz *regexp.Regexp
var yml *regexp.Regexp

type Template struct {
	Path     string
	Contents string
}

type Property struct {
	Description string
	Default     interface{}
}

type Job struct {
	Name       string
	Spec       string
	Properties map[string]Property
	Templates  []Template
}

type Release struct {
	Name    string
	Version string
	Sha1    string
	Jobs    []Job
}

type Tile struct {
	Releases []Release
	Version string
	
}

func yamldump(raw io.Reader, prefix string) {
	io.Copy(os.Stdout, prefixer.New(raw, prefix))
	/*
		b, err := ioutil.ReadAll(raw)
		if err != nil {
			log.Fatal(err)
		}

		var out interface{}
		err = yaml.Unmarshal(b, &out)
		if err != nil {
			log.Fatal(err)
		}
	*/
}

func tarball(raw io.Reader, prefix string) {
	gz, err := gzip.NewReader(raw)
	if err != nil {
		log.Fatal(err)
	}

	t := tar.NewReader(gz)
	i := 0
	for {
		header, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue

		case tar.TypeReg:
			fmt.Printf("%s- %s\n", prefix, header.Name)
			if tgz.MatchString(header.Name) {
				tarball(t, prefix+"  ")
			}
			if yml.MatchString(header.Name) {
				yamldump(t, prefix+"  ")
			}

		default:
			fmt.Printf("%s : %c %s %s\n", "Yikes! Unable to figure out type", header.Typeflag, "in file", header.Name)
		}
		i++
	}

	gz.Close()
}

func main() {
	r, err := zip.OpenReader("p-gemfire-1.6.0.0.pivotal")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	tgz = regexp.MustCompile(".tgz$")
	yml = regexp.MustCompile(".(yml|MF)$")

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		fmt.Printf(">> %s\n", f.Name)

		if tgz.MatchString(f.Name) {
			raw, err := f.Open()
			if err != nil {
				log.Fatal(err)
			}

			tarball(raw, "  ")
			raw.Close()
		}
		if yml.MatchString(f.Name) {
			raw, err := f.Open()
			if err != nil {
				log.Fatal(err)
			}

			yamldump(raw, "  ")
			raw.Close()
		}
	}
	return
}
