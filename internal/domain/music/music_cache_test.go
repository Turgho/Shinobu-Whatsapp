// Testes do cache de áudio em disco (sem rede).
package music

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestCache(t *testing.T) (*Cache, string) {
	dir := filepath.Join(t.TempDir(), "cache")
	c := NewCache(dir, nil)
	if c == nil {
		t.Fatal("cache deveria ativar com dir válido")
	}
	return c, dir
}

func TestCacheRoundTrip(t *testing.T) {
	c, _ := newTestCache(t)
	if p := c.Get("mc hariel parceiro"); p != "" {
		t.Fatal("cache deveria iniciar vazio")
	}

	dst := c.Put("mc hariel parceiro")
	if err := os.WriteFile(dst, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := c.Get("mc hariel parceiro"); got != dst {
		t.Fatalf("hit esperado em %s, got %s", dst, got)
	}

	// chave normalizada: case/whitespace não mudam a entrada
	if got := c.Get("  MC HARIEL PARCEIRO "); got != dst {
		t.Fatalf("chave deveria normalizar: %s", got)
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	c, _ := newTestCache(t)
	dst := c.Put("q")
	os.WriteFile(dst, []byte("audio"), 0o644)

	// volta o mod-time 8 dias
	old := time.Now().AddDate(0, 0, -8)
	if err := os.Chtimes(dst, old, old); err != nil {
		t.Fatal(err)
	}
	if p := c.Get("q"); p != "" {
		t.Fatal("entrada vencida deveria expirar")
	}
	// arquivo vencido é removido
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("arquivo vencido deveria ser removido")
	}
}

func TestCacheSweep(t *testing.T) {
	c, dir := newTestCache(t)
	for _, q := range []string{"a", "b", "c"} {
		p := c.Put(q)
		if err := os.WriteFile(p, []byte(q), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// vence só "b"
	old := time.Now().AddDate(0, 0, -8)
	os.Chtimes(filepath.Join(dir, cacheKey("b")), old, old)

	c.Sweep()

	if c.Get("a") == "" || c.Get("c") == "" {
		t.Fatal("entradas frescas não devem ser removidas")
	}
	if c.Get("b") != "" {
		t.Fatal("entrada vencida deveria ser removida no sweep")
	}
}

func TestCacheStatsAndDisabled(t *testing.T) {
	c, _ := newTestCache(t)
	p := c.Put("x")
	os.WriteFile(p, []byte(strings2Repeat()), 0o644)
	if n, mb := c.Stats(); n != 1 || mb <= 0 {
		t.Fatalf("stats esperadas 1/%v, got %d/%f", strings2Repeat(), n, mb)
	}

	// cache desativado: nil-safe por construção
	var dead *Cache
	if dead != nil {
		t.Fatal("cache nil deveria representar desativado")
	}
	if dead.Get("q") != "" || dead.Put("q") != "" {
		t.Fatal("métodos em nil deveriam ser no-op")
	}
}

func strings2Repeat() string {
	s := ""
	for i := 0; i < 1024; i++ {
		s += "x"
	}
	return s
}

func TestMoveFileAtomicRename(t *testing.T) {
	a := filepath.Join(t.TempDir(), "src.m4a")
	b := filepath.Join(t.TempDir(), "dst.m4a")
	os.WriteFile(a, []byte("conteudo"), 0o644)
	if err := moveFile(a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Fatal("origem deveria ter sido movida")
	}
	if data, _ := os.ReadFile(b); string(data) != "conteudo" {
		t.Fatal("conteúdo perdido no move")
	}
}
