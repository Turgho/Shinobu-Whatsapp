package ignore

import (
	"sync"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
)

const storeFile = "assets/info/ignored.json"

var (
	js      = store.NewJSONStore[[]string](storeFile)
	mu      sync.RWMutex
	ignored map[string]bool
	once    sync.Once
)

func load() {
	list, err := js.Read()
	if err != nil || list == nil {
		ignored = make(map[string]bool)
		return
	}
	ignored = make(map[string]bool, len(list))
	for _, jid := range list {
		ignored[jid] = true
	}
}

func persist() {
	list := make([]string, 0, len(ignored))
	for jid := range ignored {
		list = append(list, jid)
	}
	_ = js.Write(list)
}

func IsIgnored(jid string) bool {
	once.Do(load)

	mu.RLock()
	defer mu.RUnlock()
	return ignored[jid]
}

func Add(jid string) error {
	once.Do(load)

	mu.Lock()
	defer mu.Unlock()
	ignored[jid] = true
	persist()
	return nil
}

func Remove(jid string) (bool, error) {
	once.Do(load)

	mu.Lock()
	defer mu.Unlock()
	if !ignored[jid] {
		return false, nil
	}
	delete(ignored, jid)
	persist()
	return true, nil
}

func List() []string {
	once.Do(load)

	mu.RLock()
	defer mu.RUnlock()
	list := make([]string, 0, len(ignored))
	for jid := range ignored {
		list = append(list, jid)
	}
	return list
}
