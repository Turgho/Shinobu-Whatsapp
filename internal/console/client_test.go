package console

import (
	"encoding/json"
	"testing"
)

// O contrato com o Console precisa bater: o Console devolve "desired".
// (Bug real: o bot esperava "disabled" e recebia lista vazia.)
func TestStateDecodificaDesired(t *testing.T) {
	var st State
	if err := json.Unmarshal([]byte(`{"desired":["clima","play"],"updatedAt":"2026-11-10T00:00:00Z"}`), &st); err != nil {
		t.Fatalf("decode falhou: %v", err)
	}
	if len(st.Desired) != 2 || st.Desired[0] != "clima" {
		t.Fatalf("desired não foi decodificado: %v", st.Desired)
	}
}
