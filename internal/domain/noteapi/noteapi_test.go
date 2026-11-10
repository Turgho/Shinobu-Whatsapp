package noteapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// O noteapi precisa classificar a resposta HTTP: um 500 NÃO pode virar
// "✅ reiniciando" para o owner (foi o bug do sucesso falso).
func TestActionServidorErro(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", nil)
	_, code, err := c.Action(context.Background(), "/admin/restart", 5*time.Second)
	if err != nil {
		t.Fatalf("requisição não deveria falhar no transporte: %v", err)
	}
	if code != http.StatusInternalServerError {
		t.Errorf("código inesperado: %d", code)
	}
	if OK(code) {
		t.Error("código 500 não pode ser considerado sucesso")
	}
}

func TestActionServidorOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", nil)
	_, code, err := c.Action(context.Background(), "/admin/restart", 5*time.Second)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !OK(code) {
		t.Errorf("código %d deveria ser sucesso", code)
	}
}

// Conexão morta devolve code == 0 — quem chama decide se isso é
// "provavelmente executou" (operação destrutiva) ou falha.
func TestActionConexaoMorta(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "token", nil)
	_, code, err := c.Action(context.Background(), "/admin/reboot", 2*time.Second)
	if err == nil {
		t.Fatal("esperava erro de transporte")
	}
	if code != 0 {
		t.Errorf("conexão morta deveria devolver code 0, veio %d", code)
	}
	if OK(code) {
		t.Error("code 0 não é sucesso")
	}
}

func TestOK(t *testing.T) {
	casos := map[int]bool{
		200: true, 201: true, 204: true, 299: true,
		0: false, 301: false, 400: false, 401: false, 500: false,
	}
	for code, esperado := range casos {
		if got := OK(code); got != esperado {
			t.Errorf("OK(%d) = %v, esperava %v", code, got, esperado)
		}
	}
}
