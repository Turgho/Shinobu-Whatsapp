// Package cache é um cache em memória com TTL, para consultas externas que
// mudam pouco (clima, geocoding, cotações, feriados).
//
// É propositalmente pequeno: map + expiração por item, sem dependência
// externa e sem fundo de disco.
package cache

import (
	"sync"
	"time"
)

// Cache guarda valores por chave com validade individual.
type Cache[T any] struct {
	mu    sync.RWMutex
	items map[string]cacheEntry[T]
	ttl   time.Duration
}

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// New cria um cache com TTL padrão.
func New[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{
		items: make(map[string]cacheEntry[T]),
		ttl:   ttl,
	}
}

// Get devolve o valor em cache se ainda for válido.
func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		return zero, false
	}
	return item.value, true
}

// Set guarda um valor pelo TTL do cache.
func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry[T]{value: value, expiresAt: time.Now().Add(c.ttl)}
}

// Len devolve o número de itens (inclui expirados não removidos).
func (c *Cache[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Sweep remove os itens expirados (chamado periodicamente).
func (c *Cache[T]) Sweep() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, k)
		}
	}
}
