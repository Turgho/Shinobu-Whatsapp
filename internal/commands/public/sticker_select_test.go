package public

import "testing"

func TestPickRandomSticker_ListaVazia(t *testing.T) {
	name, ok := pickRandomSticker(nil, nil)
	if ok || name != "" {
		t.Fatalf("esperava (\"\", false), got (%q, %v)", name, ok)
	}
}

func TestPickRandomSticker_TudoNaBlacklist(t *testing.T) {
	names := []string{"ruby", "shinobu", "hitagi"}
	bl := []string{"ruby", "shinobu", "hitagi"}

	name, ok := pickRandomSticker(names, bl)
	if ok || name != "" {
		t.Fatalf("esperava (\"\", false), got (%q, %v)", name, ok)
	}
}

func TestPickRandomSticker_CasoGeral(t *testing.T) {
	names := []string{"ruby", "shinobu", "hitagi", "mayoi"}

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name, ok := pickRandomSticker(names, []string{"hitagi"})
		if !ok {
			t.Fatal("esperava true")
		}
		if name == "hitagi" {
			t.Fatalf("sticker %q não deveria ser sorteado (está na blacklist)", name)
		}
		seen[name] = true
	}

	// Com 100 iterações, espera-se que mais de 1 nome tenha sido sorteado
	if len(seen) < 2 {
		t.Fatalf("esperava diversidade de sorteio, apenas %d nomes distintos", len(seen))
	}
}
