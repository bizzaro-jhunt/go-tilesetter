package web

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/jhunt/go-tilesetter/tile"
)

type TileWrapper struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ReleaseWrapper struct {
	Tile    TileWrapper  `json:"tile"`
	Release tile.Release `json:"release"`
}

type JobWrapper struct {
	Tile    TileWrapper `json:"tile"`
	Release string      `json:"release"`
	Job     tile.Job    `json:"job"`
}

func Listen(assets, bind string) error {
	tiles := make(map[string]tile.Tile)

	/* all the UI bits are in one directory */
	http.Handle("/", http.FileServer(http.Dir(assets)))

	/*
		GET /v1/tiles
		GET /v1/tiles/:name/:version
		GET /v1/tiles/:name/:version/:release(.tgz)?
		GET /v1/tiles/:name/:version/:release/:job
	*/
	http.HandleFunc("/v1/tiles/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			w.WriteHeader(405)
			w.Write([]byte("Method not allowed.  Sorry."))
			return
		}

		parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		parts = parts[1:] // shift

		switch len(parts) {
		case 1: // GET tiles/
		case 3: // GET tiles/:name/:version
		case 4: // GET tiles/:name/:version/:release(.tgz)?
		case 5: // GET tiles/:name/:version/:release/:job
		default:
			w.WriteHeader(404)
			fmt.Fprintf(w, "That is not a valid endpoint. (%v @%v)\n", len(parts), parts)
		}

		if len(parts) == 1 {
			l := make([]string, 0)
			for k := range tiles {
				l = append(l, k)
			}
			b, err := json.Marshal(l)
			if err != nil {
				w.Header().Add("Content-type", "application/json")
				w.WriteHeader(500)
				fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
				return
			}
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(200)
			fmt.Fprintf(w, "%s\n", string(b))
			return
		}

		t, ok := tiles[fmt.Sprintf("%s/%s", parts[1], parts[2])]
		if !ok {
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(404)
			fmt.Fprintf(w, "{\"error\":\"tile %s v%s not found\"}\n", parts[1], parts[2])
			return
		}

		if len(parts) == 3 {
			b, err := json.Marshal(t)
			if err != nil {
				w.Header().Add("Content-type", "application/json")
				w.WriteHeader(500)
				fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
				return
			}
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(200)
			fmt.Fprintf(w, "%s\n", string(b))
			return
		}

		// FIXME: handle tiles/:name/:version/:release.tgz
		r, ok := t.FindRelease(parts[3])
		if !ok {
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(404)
			fmt.Fprintf(w, "{\"error\":\"release %s not found in tile %s v%s\"}\n", parts[3], t.Name, t.Version)
			return
		}

		if len(parts) == 4 {
			b, err := json.Marshal(ReleaseWrapper{
				Tile: TileWrapper{
					Name:    t.Name,
					Version: t.Version,
				},
				Release: r,
			})
			if err != nil {
				w.Header().Add("Content-type", "application/json")
				w.WriteHeader(500)
				fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
				return
			}
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(200)
			fmt.Fprintf(w, "%s\n", string(b))
			return
		}

		j, ok := r.FindJob(parts[4])
		if !ok {
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(404)
			fmt.Fprintf(w, "{\"error\":\"job %s not found in release %s v%s from tile %s v%s\"}\n", parts[4], r.Name, r.Version, t.Name, t.Version)
			return
		}

		if len(parts) == 5 {
			b, err := json.Marshal(JobWrapper{
				Tile: TileWrapper{
					Name:    t.Name,
					Version: t.Version,
				},
				Release: r.Name,
				Job:     j,
			})
			if err != nil {
				w.Header().Add("Content-type", "application/json")
				w.WriteHeader(500)
				fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
				return
			}
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(200)
			fmt.Fprintf(w, "%s\n", string(b))
			return
		}

		w.WriteHeader(500)
		fmt.Fprintf(w, "Invalid /tiles API call.\n")
		return
	})

	/*
		POST /upload
	*/
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			w.Write([]byte("Method not allowed.  Sorry."))
			return
		}

		r.ParseMultipartForm(256 << 10) // 256 M
		f, header, err := r.FormFile("tile")
		if err != nil {
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(500)
			fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
			return
		}
		fmt.Printf(">> uploading new tile (%s)\n", header.Filename)
		defer f.Close()

		tmp, err := ioutil.TempFile("", "tilesetter")
		if err != nil {
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(500)
			fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
			return
		}

		io.Copy(tmp, f)
		t, err := tile.Unpack(tmp.Name())
		if err != nil {
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(500)
			fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
			return
		}

		tiles[fmt.Sprintf("%s/%s", t.Name, t.Version)] = t
		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, "{\"ok\":\"uploaded tile %s v%s\"}\n", t.Name, t.Version)
		return
	})

	/*
		GET /v1/search?q=<query>
	*/
	http.HandleFunc("/v1/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			w.Write([]byte("Method not allowed.  Sorry."))
			return
		}

		needle := r.URL.Query().Get("q")
		props := make([]tile.MatchingProperty, 0)
		for _, t := range tiles {
			for _, m := range t.MatchProperty(needle) {
				props = append(props, m)
			}
		}

		templates := make([]tile.MatchingTemplate, 0)
		for _, t := range tiles {
			for _, m := range t.MatchTemplates(props) {
				templates = append(templates, m)
			}
		}

		data := struct {
			Properties []tile.MatchingProperty `json:"properties"`
			Templates  []tile.MatchingTemplate `json:"templates"`
		}{
			Properties: props,
			Templates:  templates,
		}

		b, err := json.Marshal(data)
		if err != nil {
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(500)
			fmt.Fprintf(w, "{\"error\":\"%s\"}\n", err)
			return
		}
		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, "%s\n", string(b))
		return
	})

	return http.ListenAndServe(bind, nil)
}
