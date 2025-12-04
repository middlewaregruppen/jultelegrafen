package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
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

type Store struct {
	entries []Entry
	mu      sync.Mutex
}

// NewStore returns a new store
func NewStore() (*Store, error) {
	f, err := os.OpenFile(DBPATH, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return &Store{}, err
}

// List lists all entries in the store
func (s *Store) List() ([]*Entry, error) {
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

// Save adds an entry to the store
func (s *Store) Save(e *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// Pop returns the first element from the entries slice. The
// popped element is then removed from the store.
func (s *Store) Pop() (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.List()
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		entries = append(entries, &Entry{Author: "Tomten", Content: RandomRim(randomContent), Created: time.Now(), IPAddr: "1270.0.0"})
	}

	// Don't try do pop an entry from an empty list
	if len(entries) < 1 {
		return nil, fmt.Errorf("can't pop entry from empty list")
	}

	popped := entries[0]
	entries = entries[1:]

	b, err := json.Marshal(&entries)
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp(".", "messages-")
	if err != nil {
		return nil, err
	}

	_, err = tmp.Write(b)
	if err != nil {
		return nil, err
	}

	err = os.Rename(tmp.Name(), DBPATH)
	if err != nil {
		return nil, err
	}

	return popped, nil
}
