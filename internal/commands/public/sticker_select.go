package public

import (
	"math/rand/v2"
	"strings"
)

// pickRandomSticker seleciona um sticker aleatório da lista,
// excluindo os que estão na blacklist. Retorna o nome e true,
// ou ("", false) se não houver stickers disponíveis.
func pickRandomSticker(names []string, blacklist []string) (string, bool) {
	bl := make(map[string]struct{}, len(blacklist))
	for _, b := range blacklist {
		bl[strings.ToLower(strings.TrimSpace(b))] = struct{}{}
	}

	candidates := make([]string, 0, len(names))
	for _, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if _, excluded := bl[key]; excluded {
			continue
		}
		candidates = append(candidates, n)
	}

	if len(candidates) == 0 {
		return "", false
	}

	return candidates[rand.IntN(len(candidates))], true
}
