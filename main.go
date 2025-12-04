package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// How long clients must wait until a new message is returned when popping
var popMaxWait time.Duration

func init() {
	pflag.DurationVarP(&popMaxWait, "pop-max-wait", "p", time.Minute*2, "How long clients must wait until a new message is returnes when popping")
}

type Entry struct {
	Author  string    `json:"author,omitempty"`
	Content string    `json:"content,omitempty"`
	Created time.Time `json:"created,omitempty"`
	IPAddr  string    `json:"ip_addr,omitempty"`
}

type APIResult struct {
	Entries []Entry `json:"entries"`
}

const DBPATH string = "messages.json"

//go:embed static/*
var staticFS embed.FS

func main() {
	mux := http.NewServeMux()

	pflag.Parse()

	store, err := NewStore(popMaxWait)
	if err != nil {
		log.Fatalf("error opening store: %v", err)
	}

	// Serve the SPA index for root and any front‑end route
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// For any path that doesn't look like a static asset, serve index.html
		if r.URL.Path == "/" || path.Ext(r.URL.Path) == "" {
			f, err := fs.Sub(staticFS, "static")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusInternalServerError)
				return
			}
			http.FileServer(http.FS(f)).ServeHTTP(w, r)
			return
		}
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/message", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			if err := handleGet(store, r, w); err != nil {
				log.Printf("get handler returned error: %v", err)
			}
		case http.MethodPost:
			if err := handlePost(store, r, w); err != nil {
				log.Printf("post handler returned error: %v", err)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/message/pop", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if err := handlePop(store, r, w); err != nil {
				log.Printf("pop handler returned error: %v", err)
			}
		}
	})
	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handlePop(store *Store, _ *http.Request, w http.ResponseWriter) error {
	popped, err := store.Pop()
	if err != nil {
		http.Error(w, "error listing entries", http.StatusInternalServerError)
		return err
	}

	b, err := json.Marshal(popped)
	if err != nil {
		http.Error(w, "error marshalling popped entry", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		return err
	}

	return nil
}

// Handles get requests
func handleGet(store *Store, _ *http.Request, w http.ResponseWriter) error {
	entries, err := store.List()
	if err != nil {
		http.Error(w, "error listing entries", http.StatusInternalServerError)
		return err
	}

	b, err := json.Marshal(&entries)
	if err != nil {
		http.Error(w, "error marshalling entries", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		return err
	}

	return nil
}

// Handles post requests
func handlePost(store *Store, r *http.Request, w http.ResponseWriter) error {
	var req Entry
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return err
	}

	// Validation
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return nil
	}
	if strings.TrimSpace(req.Author) == "" {
		http.Error(w, "author is required", http.StatusBadRequest)
		return nil
	}
	if len(req.Content) > 100 {
		http.Error(w, "message is too long. max len is 100", http.StatusBadRequest)
		return nil
	}

	// Set metadata before commiting to store
	req.Created = time.Now()
	req.IPAddr = r.RemoteAddr

	err := store.Save(&req)
	if err != nil {
		http.Error(w, "error saving to store", http.StatusInternalServerError)
		return err
	}

	entries, err := store.List()
	if err != nil {
		http.Error(w, "error listing entries", http.StatusInternalServerError)
		return err
	}

	b, err := json.Marshal(&entries)
	if err != nil {
		http.Error(w, "error marshalling entries", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		return err
	}

	return nil
}
