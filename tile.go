package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"gopkg.in/yaml.v2"
)

var (
	JobTarballPattern      *regexp.Regexp
	JobSpecFilePattern     *regexp.Regexp
	JobTemplateFilePattern *regexp.Regexp

	ReleaseTarballPattern  *regexp.Regexp
	ReleaseSpecFilePattern *regexp.Regexp

	TileSpecFilePattern *regexp.Regexp
)

func init() {
	JobTarballPattern = regexp.MustCompile("^./jobs/(.*)\\.tgz$")
	JobSpecFilePattern = regexp.MustCompile("^./job.MF$")
	JobTemplateFilePattern = regexp.MustCompile("^./templates/(.*)$")

	ReleaseTarballPattern = regexp.MustCompile("^releases/(.*)\\.tgz$")
	ReleaseSpecFilePattern = regexp.MustCompile("^\\./release.MF$")

	TileSpecFilePattern = regexp.MustCompile("^metadata/tile_metadata\\.yml$")
}

var tgz *regexp.Regexp
var yml *regexp.Regexp

type Template struct {
	Path     string
	Contents string
}

type Property struct {
	Description string
	Default     interface{} `yaml:"default,omitempty"`
}

type Job struct {
	Name       string
	Spec       string `yaml:"spec,omitempty"`
	Properties map[string]Property
	Templates  []Template
}

type Release struct {
	Name    string
	Spec    string `yaml:"spec,omitempty"`
	Version string
	Sha1    string
	Jobs    []Job
}

type Tile struct {
	Name     string
	Spec     string `yaml:"spec,omitempty"`
	Releases []Release
	Version  string
}

func includeSpec() bool {
	return os.Getenv("NOSPEC") == ""
}

func unpackTile(path string) (Tile, error) {
	var t Tile

	z, err := zip.OpenReader(path)
	if err != nil {
		return t, err
	}
	defer z.Close()

	var spec []byte
	for _, f := range z.File {
		if TileSpecFilePattern.MatchString(f.Name) {
			raw, err := f.Open()
			if err != nil {
				return t, err
			}
			spec, err = ioutil.ReadAll(raw)
			if err != nil {
				return t, err
			}

		} else if ReleaseTarballPattern.MatchString(f.Name) {
			raw, err := f.Open()
			if err != nil {
				return t, err
			}
			r, err := unpackRelease(raw)
			if err != nil {
				return t, err
			}
			t.Releases = append(t.Releases, r)
		}
	}

	if len(spec) == 0 {
		return t, fmt.Errorf("no spec file found for tile")
	}

	var d map[string]interface{}
	err = yaml.Unmarshal(spec, &d)
	if err != nil {
		return t, nil
	}

	if name, ok := d["name"]; ok {
		t.Name = name.(string)
	} else {
		return t, fmt.Errorf("no name present in tile spec")
	}

	if version, ok := d["product_version"]; ok {
		t.Version = version.(string)
	} else {
		return t, fmt.Errorf("no product version present in tile spec")
	}

	if includeSpec() {
		t.Spec = string(spec)
	}
	return t, nil
}

func unpackRelease(f io.ReadCloser) (Release, error) {
	var r Release

	tarball, err := ioutil.ReadAll(f)
	if err != nil {
		return r, err
	}
	f.Close()

	r.Sha1 = fmt.Sprintf("%x", sha1.Sum(tarball))

	gz, err := gzip.NewReader(bytes.NewBuffer(tarball))
	if err != nil {
		return r, err
	}

	var spec []byte
	t := tar.NewReader(gz)
	for {
		header, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return r, err
		}

		if header.Typeflag == tar.TypeReg {
			if ReleaseSpecFilePattern.MatchString(header.Name) {
				spec, err = ioutil.ReadAll(t)
				if err != nil {
					return r, err
				}

			} else if JobTarballPattern.MatchString(header.Name) {
				j, err := unpackJob(t)
				if err != nil {
					return r, err
				}
				r.Jobs = append(r.Jobs, j)
			}
		}
	}

	if len(spec) == 0 {
		return r, fmt.Errorf("no spec file found for release")
	}

	var d map[string]interface{}
	err = yaml.Unmarshal(spec, &d)
	if err != nil {
		return r, nil
	}

	if name, ok := d["name"]; ok {
		r.Name = name.(string)
	} else {
		return r, fmt.Errorf("no name present in release spec")
	}

	if version, ok := d["version"]; ok {
		r.Version = version.(string)
	} else {
		return r, fmt.Errorf("no version present in release spec")
	}

	if includeSpec() {
		r.Spec = string(spec)
	}
	root := os.Getenv("TARBALLS")
	if root != "" {
		err = ioutil.WriteFile(fmt.Sprintf("%s/%s-%s.tgz", root, r.Name, r.Version), tarball, 0666)
	}
	return r, nil
}

func unpackJob(f io.Reader) (Job, error) {
	var j Job
	tpl := make(map[string]string)

	gz, err := gzip.NewReader(f)
	if err != nil {
		return j, err
	}

	var spec []byte
	t := tar.NewReader(gz)
	for {
		header, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return j, err
		}

		if header.Typeflag == tar.TypeReg {
			if JobSpecFilePattern.MatchString(header.Name) {
				spec, err = ioutil.ReadAll(t)
				if err != nil {
					return j, err
				}

			} else if JobTemplateFilePattern.MatchString(header.Name) {
				l := JobTemplateFilePattern.FindStringSubmatch(header.Name)
				b, err := ioutil.ReadAll(t)
				if err != nil {
					return j, err
				}
				tpl[l[1]] = string(b)
			}
		}
	}

	if len(spec) == 0 {
		return j, fmt.Errorf("no spec file found for job")
	}

	var d map[string]interface{}
	err = yaml.Unmarshal(spec, &d)
	if err != nil {
		return j, nil
	}

	if name, ok := d["name"]; ok {
		j.Name = name.(string)
	} else {
		return j, fmt.Errorf("no name present in job spec")
	}

	j.Properties = make(map[string]Property)
	if _props, ok := d["properties"]; ok {
		for k, _v := range _props.(map[interface{}]interface{}) {
			v := _v.(map[interface{}]interface{})
			p := Property{}
			if x, ok := v["description"]; ok {
				p.Description = fmt.Sprintf("%s", x)
			}
			if x, ok := v["default"]; ok {
				p.Default = x
			}
			j.Properties[fmt.Sprintf("%s", k)] = p
		}
	}

	if _tpl, ok := d["templates"]; ok {
		for k, v := range _tpl.(map[interface{}]interface{}) {
			t := Template{}
			t.Path = fmt.Sprintf("/var/vcap/jobs/%s/%s", j.Name, v)
			t.Contents = tpl[k.(string)]
			j.Templates = append(j.Templates, t)
		}
	}

	if includeSpec() {
		j.Spec = string(spec)
	}
	return j, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s path/to/tile.pivotal\n", os.Args[0])
		os.Exit(1)
	}

	fmt.Printf("# %s\n", os.Args[1])
	t, err := unpackTile(os.Args[1])
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
