package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type JSONStore[T any] struct {
	path string
	mu   sync.RWMutex
}

func NewJSONStore[T any](path string) *JSONStore[T] {
	return &JSONStore[T]{path: path}
}

func (s *JSONStore[T]) Read() (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var zero T
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return zero, nil
	}
	if err != nil {
		return zero, fmt.Errorf("ler arquivo %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return zero, nil
	}
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return zero, fmt.Errorf("parse json %s: %w", s.path, err)
	}
	return val, nil
}

func (s *JSONStore[T]) Write(val T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeLocked(val)
}

func (s *JSONStore[T]) Update(fn func(T) (T, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, err := s.readLocked()
	if err != nil {
		return err
	}

	newVal, err := fn(val)
	if err != nil {
		return err
	}

	return s.writeLocked(newVal)
}

func (s *JSONStore[T]) readLocked() (T, error) {
	var zero T
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return zero, nil
	}
	if err != nil {
		return zero, fmt.Errorf("ler arquivo %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return zero, nil
	}
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return zero, fmt.Errorf("parse json %s: %w", s.path, err)
	}
	return val, nil
}

// writeLocked escreve no disco de forma atômica: primeiro cria um .tmp, depois renomeia.
// Isso evita que um crash no meio da escrita corrompa o arquivo original.
func (s *JSONStore[T]) writeLocked(val T) error {
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("escrever temporário: %w", err)
	}
	// Rename é atômico no Linux (mesmo filesystem): o arquivo original
	// só é substituído quando o .tmp já foi escrito por completo.
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("renomear: %w", err)
	}
	return nil
}
