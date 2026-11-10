// Testes dos helpers de geocoding — sem rede real (httptest local).
package geocoding

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestCombineDisplayName(t *testing.T) {
	cases := []struct {
		city, state, fallback, want string
	}{
		{"Goiânia", "Goiás", "", "Goiânia, Goiás"},
		{"Goiânia", "", "", "Goiânia"},       // estado vazio: sem vírgula pendente
		{"", "Goiás", "Faro", "Goiás"},       // só estado (bug do Nominatim)
		{"", "", "Faro", "Faro"},             // fallback
		{"  Rio  ", "  RJ  ", "", "Rio, RJ"}, // trim das partes
	}
	for _, c := range cases {
		if got := combineDisplayName(c.city, c.state, c.fallback); got != c.want {
			t.Errorf("combineDisplayName(%q, %q, %q) = %q, want %q",
				c.city, c.state, c.fallback, got, c.want)
		}
	}
}

func TestResolveGeoJSON(t *testing.T) {
	log := zap.NewNop()
	g := NewGeoCoding("http://x", "http://y", log)

	// 200 + JSON válido
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results":[{"name":"Goiânia"}]}`))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := g.resolveGeoJSON(resp, &out, "nominatim"); err != nil {
		t.Fatalf("esperava sucesso: %v", err)
	}
	if len(out.Results) != 1 || out.Results[0].Name != "Goiânia" {
		t.Fatalf("payload inesperado: %+v", out)
	}

	// 403 → erro com status (mesma categoria usada pelos dois lookups)
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv2.Close()

	resp2, err := http.Get(srv2.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	err = g.resolveGeoJSON(resp2, &struct{}{}, "open-meteo")
	if err == nil {
		t.Fatal("esperava erro em status 403")
	}
	if err.Error() != "geocoding: status 403" {
		t.Fatalf("esperava 'geocoding: status 403', got: %v", err)
	}
}
