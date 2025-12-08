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
	pflag.DurationVarP(&popMaxWait, "pop-max-wait", "p", time.Minute*2, "How long clients must wait until a new message is returnes when popping. Value 0 means pop is instantaneous")
}

type APIResult struct {
	Entries []Entry `json:"entries"`
}

const (
	DBPATH    string = "messages.json"
	QUEUEPATH string = "queue.json"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	mux := http.NewServeMux()

	pflag.Parse()

	store, err := NewStore(
		WithFilePath(DBPATH),
		WithPopMaxWait(popMaxWait),
	)
	if err != nil {
		log.Fatalf("error opening store: %v", err)
	}
	if err != nil {
		log.Fatalf("error opening queue: %v", err)
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

	mux.HandleFunc("/api/queue", func(w http.ResponseWriter, r *http.Request) {
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

	mux.HandleFunc("/api/queue/pop", func(w http.ResponseWriter, r *http.Request) {
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
	entries, err := store.Queue()
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
		http.Error(w, "En julhälsning utan meddelande är som risgrynsgröt utan kanel", http.StatusBadRequest)
		return nil
	}
	if strings.TrimSpace(req.Author) == "" {
		http.Error(w, "Jag vill gärna veta vad du heter", http.StatusBadRequest)
		return nil
	}
	if len(req.Content) > 100 {
		http.Error(w, "Det var lite mer än vad jag klarar av. Va snäll å korta ner meddelandet lite", http.StatusBadRequest)
		return nil
	}
	if len(req.Author) > 22 {
		http.Error(w, "Jag fixar inte namn som är längre än 22 tecken. Fråga mig inte varför", http.StatusBadRequest)
		return nil
	}

	req.IPAddr = r.RemoteAddr
	req.UserAgent = r.UserAgent()

	e, err := store.Save(&req)
	if err != nil {
		http.Error(w, "error saving to store", http.StatusInternalServerError)
		return err
	}

	b, err := json.Marshal(e)
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
