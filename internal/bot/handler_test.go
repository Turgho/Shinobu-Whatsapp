package bot

import "testing"

// O bot pode ficar "rodando" (processo vivo) e mesmo assim mudo, quando o
// handshake falha depois do Connect() retornar nil (ex.: 405 Client
// outdated). O dono precisa ser avisado — uma vez por episode, sem spam
// do auto-reconnect.
func TestMarkConnectFailedAvisaUmaVez(t *testing.T) {
	h := &Handler{}

	if !h.markConnectFailed() {
		t.Fatal("a primeira falha deveria gerar aviso")
	}
	if h.markConnectFailed() {
		t.Error("a segunda falha (auto-reconnect) não deveria gerar novo aviso")
	}
	if h.markConnectFailed() {
		t.Error("a terceira falha não deveria gerar novo aviso")
	}
}

// Depois de reconectar, uma nova falha deve gerar aviso de novo.
func TestMarkConnectedMarcaRecuperacao(t *testing.T) {
	h := &Handler{}

	// Conexão inicial (sem falha anterior) não é recuperação.
	if h.markConnected() {
		t.Error("conexão inicial não deveria contar como recuperação")
	}

	h.markConnectFailed()
	if !h.markConnected() {
		t.Error("conectar depois de uma falha deveria contar como recuperação")
	}
	if h.markConnected() {
		t.Error("conectar de novo não deveria contar como recuperação")
	}

	// novo episode de falha → novo aviso
	if !h.markConnectFailed() {
		t.Error("nova falha após recuperação deveria gerar aviso")
	}
}
