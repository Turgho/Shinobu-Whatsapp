package commands

import (
	"path/filepath"
	"testing"
)

// O estado precisa sobreviver a um restart (é JSON em disco) e o Apply
// (usado na sincronização com o Console) só deve gravar quando muda.
func TestCommandStatePersiste(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disabled_commands.json")

	s := NewCommandStateAt(path)
	if s.IsDisabled("clima") {
		t.Fatal("comando novo não deveria estar desabilitado")
	}

	if err := s.SetDisabled("clima", true); err != nil {
		t.Fatalf("SetDisabled falhou: %v", err)
	}
	if !s.IsDisabled("clima") {
		t.Error("clima deveria estar desabilitado")
	}

	// nova instância (simula restart) lê o arquivo
	s2 := NewCommandStateAt(path)
	if !s2.IsDisabled("clima") {
		t.Error("estado não sobreviveu ao restart")
	}

	if err := s2.SetDisabled("clima", false); err != nil {
		t.Fatalf("SetDisabled(false) falhou: %v", err)
	}
	if s2.IsDisabled("clima") {
		t.Error("clima deveria estar habilitado de novo")
	}
}

func TestCommandStateApply(t *testing.T) {
	s := NewCommandStateAt(filepath.Join(t.TempDir(), "state.json"))

	changed, err := s.Apply([]string{"clima", "play"})
	if err != nil {
		t.Fatalf("Apply falhou: %v", err)
	}
	if !changed {
		t.Error("Apply deveria reportar mudança")
	}
	if !s.IsDisabled("clima") || !s.IsDisabled("play") {
		t.Error("comandos do Apply não foram aplicados")
	}

	// mesmo conjunto → sem mudança (não reescreve o arquivo)
	changed, err = s.Apply([]string{"play", "clima"})
	if err != nil {
		t.Fatalf("Apply falhou: %v", err)
	}
	if changed {
		t.Error("Apply com o mesmo conjunto não deveria reportar mudança")
	}

	// conjunto menor → remove
	changed, err = s.Apply([]string{"clima"})
	if err != nil {
		t.Fatalf("Apply falhou: %v", err)
	}
	if !changed {
		t.Error("Apply com conjunto menor deveria reportar mudança")
	}
	if s.IsDisabled("play") {
		t.Error("play deveria ter sido reabilitado")
	}
}

func TestCommandStateDisabled(t *testing.T) {
	s := NewCommandStateAt(filepath.Join(t.TempDir(), "state.json"))
	if err := s.SetDisabled("b", true); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDisabled("a", true); err != nil {
		t.Fatal(err)
	}

	list := s.Disabled()
	if len(list) != 2 {
		t.Fatalf("esperava 2 desabilitados, veio %d", len(list))
	}
}
