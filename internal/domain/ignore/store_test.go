package ignore

import (
	"path/filepath"
	"testing"
)

// O !ignorar grava o número e a checagem usava o JID — com endereçamento
// LID ninguém era bloqueado. Agora a chave é o número normalizado.
func TestIsIgnoredAceitaNumeroEJID(t *testing.T) {
	s := NewStoreAt(filepath.Join(t.TempDir(), "ignored.json"))
	// simula a lista já persistida com o número
	s.once.Do(func() {})
	s.ignored = map[string]string{key("5511999999999"): "5511999999999"}

	if !s.IsIgnored("5511999999999") {
		t.Error("número ignorado não foi reconhecido pelo número")
	}
	if !s.IsIgnored("5511999999999@lid") {
		t.Error("número ignorado não foi reconhecido pelo JID")
	}
	if !s.IsIgnored("5511999999999@s.whatsapp.net") {
		t.Error("número ignorado não foi reconhecido pelo JID @s.whatsapp.net")
	}
	if s.IsIgnored("5511888888888") {
		t.Error("outro contato não deveria estar ignorado")
	}
}

func TestAddRemove(t *testing.T) {
	s := NewStoreAt(filepath.Join(t.TempDir(), "ignored.json"))

	if err := s.Add("5511999999999"); err != nil {
		t.Fatalf("Add falhou: %v", err)
	}
	if !s.IsIgnored("5511999999999@lid") {
		t.Error("contato adicionado não é reconhecido pelo JID")
	}

	removido, err := s.Remove("5511999999999@lid")
	if err != nil {
		t.Fatalf("Remove falhou: %v", err)
	}
	if !removido {
		t.Error("Remove deveria informar que removeu")
	}
	if s.IsIgnored("5511999999999") {
		t.Error("contato removido ainda aparece como ignorado")
	}
}

func TestListMantemValorOriginal(t *testing.T) {
	s := NewStoreAt(filepath.Join(t.TempDir(), "ignored.json"))
	if err := s.Add("5511999999999@lid"); err != nil {
		t.Fatal(err)
	}
	list := s.List()
	if len(list) != 1 || list[0] != "5511999999999@lid" {
		t.Errorf("lista deveria manter o valor original: %v", list)
	}
}
