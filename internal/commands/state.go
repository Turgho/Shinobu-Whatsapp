package commands

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
)

// defaultStateFile é onde os comandos desabilitados são persistidos.
// Mesmo diretório da lista de ignorados (assets/info).
const defaultStateFile = "assets/info/disabled_commands.json"

// CommandState guarda quais comandos estão desabilitados.
//
// Serve para tirar um comando específico de circulation sem derrubar o
// bot inteiro: um comando com bug é desabilitado, o resto continua
// respondendo. O estado é persistido em JSON, então sobrevive a restart.
type CommandState struct {
	js *store.JSONStore[[]string]

	mu       sync.RWMutex
	disabled map[string]bool
	once     sync.Once
}

// NewCommandState cria o estado com o arquivo padrão.
func NewCommandState() *CommandState {
	return NewCommandStateAt(defaultStateFile)
}

// NewCommandStateAt cria o estado com um caminho explícito (testes).
func NewCommandStateAt(path string) *CommandState {
	return &CommandState{
		js:       store.NewJSONStore[[]string](path),
		disabled: make(map[string]bool),
	}
}

func (s *CommandState) load() {
	list, err := s.js.Read()
	if err != nil || list == nil {
		s.disabled = make(map[string]bool)
		return
	}
	s.disabled = make(map[string]bool, len(list))
	for _, name := range list {
		if name != "" {
			s.disabled[name] = true
		}
	}
}

// persist grava a lista de desabilitados.
func (s *CommandState) persist() error {
	list := make([]string, 0, len(s.disabled))
	for name := range s.disabled {
		list = append(list, name)
	}
	if dir := filepath.Dir(s.js.Path()); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return s.js.Write(list)
}

// IsDisabled informa se o comando está desabilitado.
func (s *CommandState) IsDisabled(name string) bool {
	if s == nil {
		return false
	}
	s.once.Do(s.load)

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disabled[name]
}

// SetDisabled habilita ou desabilita um comando e persiste.
func (s *CommandState) SetDisabled(name string, disabled bool) error {
	if s == nil {
		return nil
	}
	s.once.Do(s.load)

	s.mu.Lock()
	defer s.mu.Unlock()

	if disabled {
		s.disabled[name] = true
	} else {
		delete(s.disabled, name)
	}
	return s.persist()
}

// Disabled devolve a lista de comandos desabilitados.
func (s *CommandState) Disabled() []string {
	if s == nil {
		return nil
	}
	s.once.Do(s.load)

	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]string, 0, len(s.disabled))
	for name := range s.disabled {
		list = append(list, name)
	}
	return list
}

// Apply substitui o conjunto desabilitado pelo desejado (usado na
// sincronização com o Console). Retorna true se algo mudou.
func (s *CommandState) Apply(disabled []string) (bool, error) {
	if s == nil {
		return false, nil
	}
	s.once.Do(s.load)

	wanted := make(map[string]bool, len(disabled))
	for _, name := range disabled {
		if name != "" {
			wanted[name] = true
		}
	}

	s.mu.Lock()
	changed := len(wanted) != len(s.disabled)
	if !changed {
		for name := range wanted {
			if !s.disabled[name] {
				changed = true
				break
			}
		}
	}
	if changed {
		s.disabled = wanted
	}
	s.mu.Unlock()

	if !changed {
		return false, nil
	}
	return true, s.persist()
}
