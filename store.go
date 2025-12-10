package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

var randomContent []string = []string{
	// "När mörkret är stort och vintern är kall, så sprider denna värme i varje liten sal.",
	// "Ett paket så mjukt, du kan nog gissa, något du gärna vill runt kroppen ha och krama och krissa.",
	// "När batteriet dör och du suckar \"nej!\", då räddar denna hjälte snabbt din grej.",
	// "Till mys, till film och kalla kvällar, något som värmer bättre än alla fjällar.",
	// "När du vill veta tid och rum, så hjälper denna – tick tack, hum hum.",
	// "I detta kan du skriva ner stort som smått, hemligheter, tankar och allt du tänkt på gott.",
	// "När doften sprids i huset så fint, då vet man att julen är riktigt i print.",
	// "För huvudet trött och kroppen matt, ger denna vila, det är väl glatt?",
	// "När du vill lyssna och slippa sladd, är denna gåva både smart och ball.",
	// "När du fryser om händerna och huttrar så, kan denna värma – du vet nog vad ändå.",
	// "En liten sak som blinkar och lyser, perfekt när kvällens mörker smyger och fryser.",
	// "När kaffet ropar: “drick mig nu!”, är denna din vän, ja det vet du ju.",
	// "Till pyssel och knep när du vill skapa, något som hjälper att klippa och kapa.",
	// "Du scrollar och klickar varenda dag, så här får du något som ger bättre tag.",
	// "När du vill laga, steka, fixa, hjälper denna dig att inte bränna mixa.",
	// "Ett paket med spänning, kanske lite skratt, något att läsa när du vill ha det glatt.",
	// "När kroppen vill ner i vila och ro, hjälper denna dig att sova som en go.",
	// "När du springer, hoppar och tränar på, ser denna hur duktig du är ändå.",
	// "Till något som ofta försvinner därhemma, nu får du ett helt eget — lätt att glömma!",
	// "En låda med smaker, kryddiga ting, perfekt för den som älskar gott kring julens ring.",
	"När klustret skalar och trafiken brusar, ser K8s till att julens tjänster susar.",
}

func RandomRim(rim []string) string {
	if len(rim) == 0 {
		return ""
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := r.Intn(len(rim))
	return rim[index]
}

type EntryID string

type Store struct {
	entries    map[EntryID]*Entry
	mu         sync.Mutex
	lastPop    time.Time
	lastEntry  *Entry
	popMaxWait time.Duration
	filePath   string
	fileName   string
}

type Entry struct {
	ID        string `json:"id,omitempty"`
	Author    string `json:"author,omitempty"`
	Content   string `json:"content,omitempty"`
	Created   string `json:"created,omitempty"`
	IPAddr    string `json:"ip_addr,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

type StoreOpt func(*Store)

// WithFilePath provides the full path to the store file
func WithFilePath(filePath string) StoreOpt {
	return func(s *Store) {
		s.filePath = filePath
	}
}

// WithPopMaxWait sets the rate of popped messages
func WithPopMaxWait(d time.Duration) StoreOpt {
	return func(s *Store) {
		s.popMaxWait = d
	}
}

// NewStore returns a new store
func NewStore(opts ...StoreOpt) (*Store, error) {
	store := &Store{
		filePath:   "messages.json",
		popMaxWait: 0,
		entries:    map[EntryID]*Entry{},
	}

	// Apply options
	for _, opt := range opts {
		opt(store)
	}

	_, err := os.Stat(store.filePath)
	if err != nil {
		// Create empty file store if none exists
		if os.IsNotExist(err) {

			f, err := os.Create(store.filePath)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			entries := make(map[EntryID]*Entry)
			b, err := json.Marshal(&entries)
			if err != nil {
				return nil, err
			}

			_, err = f.Write(b)
			if err != nil {
				return nil, err
			}

		} else {
			return nil, err
		}
	}

	store.fileName = filepath.Base(store.filePath)

	return store, nil
}

// Queue lists all entries in the queue
func (s *Store) Queue() (map[EntryID]*Entry, error) {
	return s.entries, nil
}

// List lists all entries in the store
func (s *Store) List() (map[EntryID]*Entry, error) {
	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var result map[EntryID]*Entry
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Save adds an entry to the store
func (s *Store) Save(e *Entry) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set metadata before commiting to store
	e.Created = time.Now().String()
	e.ID = uuid.New().String()

	entries, err := s.List()
	if err != nil {
		return nil, err
	}

	entries[EntryID(e.ID)] = e
	s.entries[EntryID(e.ID)] = e

	b, err := json.Marshal(&entries)
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp(".", fmt.Sprintf("%s-", s.fileName))
	if err != nil {
		return nil, err
	}

	_, err = tmp.Write(b)
	if err != nil {
		return nil, err
	}

	err = os.Rename(tmp.Name(), s.filePath)
	if err != nil {
		return nil, err
	}

	return s.Get(e.ID)
}

// Get returns one entry with matching id
func (s *Store) Get(id string) (*Entry, error) {
	entries, err := s.List()
	if err != nil {
		return nil, err
	}

	entry, ok := entries[EntryID(id)]
	if !ok {
		return nil, os.ErrNotExist
	}

	return entry, nil
}

// Pop returns the first element from the entries slice. The
// popped element is then removed from the store. Pop has an internal backoff mechanism.
// To prevent too many concurrent pops. Only the last entry will be return within a given
// period of time.
func (s *Store) Pop() (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaultEntry := &Entry{Author: "Tomten", Content: RandomRim(randomContent), Created: time.Now().String(), IPAddr: "1270.0.0"}

	// Don't pop a new message if not enough time has passed
	since := time.Since(s.lastPop)
	if since < s.popMaxWait && s.lastEntry != nil {
		return s.lastEntry, nil
	}

	// If no messages in queue, then return a default entry until someone submits one
	if len(s.entries) == 0 {
		return defaultEntry, nil
	}

	keys := make([]EntryID, 0, len(s.entries))
	for k := range s.entries {
		keys = append(keys, k)
	}
	poppedKey := keys[0]
	popped := s.entries[poppedKey]
	delete(s.entries, poppedKey)

	s.lastPop = time.Now()
	s.lastEntry = popped
	return popped, nil
}
