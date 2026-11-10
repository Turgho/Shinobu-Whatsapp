package cache

import (
	"testing"
	"time"
)

func TestGetSet(t *testing.T) {
	c := New[string](time.Minute)

	if _, ok := c.Get("chave"); ok {
		t.Error("cache vazio não deveria ter o item")
	}

	c.Set("chave", "valor")
	got, ok := c.Get("chave")
	if !ok || got != "valor" {
		t.Errorf("Get devolveu %q (%v)", got, ok)
	}
}

func TestExpira(t *testing.T) {
	c := New[string](20 * time.Millisecond)
	c.Set("chave", "valor")

	time.Sleep(40 * time.Millisecond)
	if _, ok := c.Get("chave"); ok {
		t.Error("item expirado não deveria ser devolvido")
	}
}

func TestSweepRemoveExpirados(t *testing.T) {
	c := New[string](20 * time.Millisecond)
	c.Set("a", "1")
	c.Set("b", "2")

	time.Sleep(40 * time.Millisecond)
	c.Sweep()

	if c.Len() != 0 {
		t.Errorf("Sweep deveria remover os expirados, restou %d", c.Len())
	}
}
