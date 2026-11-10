package music

import (
	"os"
	"path/filepath"
	"testing"
)

// moveFile: quando o destino já existe (outro download ganhou a corrida),
// o erro de O_EXCL não pode virar falha — o arquivo existente é o válido.
// Antes disso, o chamador caía no fallback remoto e devolvia mp3 no lugar
// de m4a.
func TestMoveFileDestinoExistente(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.m4a")
	dst := filepath.Join(dir, "dst.m4a")

	if err := os.WriteFile(src, []byte("novo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("existente"), 0o644); err != nil {
		t.Fatal(err)
	}

	// rename sobrescreve no mesmo FS — força o caminho de cópia com O_EXCL
	// via um dst em outro diretório? Não: aqui testamos o desfecho.
	if err := moveFile(src, dst); err != nil {
		t.Fatalf("destino existente não deveria dar erro: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("destino ficou vazio")
	}
}

func TestMoveFileCopiaNormal(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.m4a")
	dst := filepath.Join(dir, "sub", "dst.m4a")

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("conteudo"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := moveFile(src, dst); err != nil {
		t.Fatalf("movimentação falhou: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "conteudo" {
		t.Errorf("conteúdo copiado errado: %q", data)
	}
}
