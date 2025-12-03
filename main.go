package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

type Entry struct {
	Author  string   `json:"author,omitempty"`
	Content string   `json:"content,omitempty"`
	Created string   `json:"created,omitempty"`
	IPAddr  net.Addr `json:"ip_addr,omitempty"`
}

type APIResult struct {
	Entries []Entry `json:"entries"`
}

const DBPATH string = "messages.json"

//go:embed static/*
var staticFS embed.FS

func main() {
	mux := http.NewServeMux()

	store, err := NewStore()
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
			// http.ServeContent(w, r, "index.html" /*modTime*/ /*zero*/, 0, f)
			http.FileServer(http.FS(f)).ServeHTTP(w, r)
			// http.FileServer(http.FS(f))
			return
		}
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
	})

	// // Example JSON API endpoint
	// mux.HandleFunc("/api/message", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.Write([]byte(`{"message": "Hello from Go + Alpine!"}`))
	// })

	// JSON API endpoint for getting/setting the message.
	mux.HandleFunc("/api/message", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			// if err := json.NewEncoder(w).Encode(messagePayload{Message: getMessage()}); err != nil {
			// 	log.Printf("error encoding GET /api/message response: %v", err)
			// }
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message": "Hello from Go + Alpine!"}`))
		case http.MethodPost:
			var req Entry
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(req.Content) == "" {
				http.Error(w, "message is required", http.StatusBadRequest)
				return
			}

			err = store.Save(&req)
			if err != nil {
				http.Error(w, "error saving to store", http.StatusInternalServerError)
				log.Printf("error saving to store:%v", err)
			}

			entries, err := store.List()
			if err != nil {
				http.Error(w, "error listing entries", http.StatusInternalServerError)
				log.Printf("error listing entries from: %v", err)
			}

			b, err := json.Marshal(&entries)
			if err != nil {
				http.Error(w, "error marshalling entries", http.StatusInternalServerError)
				log.Printf("error marshalling entries from: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(b)

			// if err := setMessage(req.Content); err != nil {
			// 	log.Printf("error saving message: %v", err)
			// 	http.Error(w, "could not save message", http.StatusInternalServerError)
			// 	return
			// }
			// if err := json.NewEncoder(w).Encode(messagePayload{Message: getMessage()}); err != nil {
			// 	log.Printf("error encoding POST /api/message response: %v", err)
			// }
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

type Store struct {
	entries []Entry
	mu      sync.Mutex
}

func NewStore() (*Store, error) {
	f, err := os.OpenFile(DBPATH, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return nil, err
}

// func (s *Store) Open(filePath string) (*os.File, error) {
// 	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0o644)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return f, nil
// }

func (s *Store) List() ([]*Entry, error) {
	// f, err := s.Open(DBPATH)
	// if err != nil {
	// 	return nil, err
	// }
	// defer f.Close()

	b, err := os.ReadFile(DBPATH)
	if err != nil {
		return nil, err
	}

	var result []*Entry
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Store) Save(e *Entry) error {
	entries, err := s.List()
	if err != nil {
		return err
	}

	entries = append(entries, e)

	b, err := json.Marshal(&entries)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(".", "messages-")
	if err != nil {
		return err
	}

	_, err = tmp.Write(b)
	if err != nil {
		return err
	}

	return os.Rename(tmp.Name(), DBPATH)
}

// func saveToFile(file string, d []byte) error {
// 	b, err := os.ReadFile(DBPATH)
// 	if err != nil {
// 		return err
// 	}
//
// 	var entries []Entry
// 	err = json.Unmarshal(b, &entries)
// 	if err != nil {
// 		return err
// 	}
//
// 	return nil
// }
