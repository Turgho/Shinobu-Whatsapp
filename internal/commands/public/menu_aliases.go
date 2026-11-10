package public

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
)

// formatAliases monta a linha de atalhos do menu (limitada para caber na
// mensagem). Só entram aliases de comandos visíveis no menu.
func formatAliases(r *commands.Router, grouped map[commands.CommandType][]commands.CommandMeta, includePrivate bool) string {
	visiveis := make(map[string]bool)
	for _, metas := range grouped {
		for _, m := range metas {
			visiveis[m.Name] = true
		}
	}

	aliases := r.Aliases()
	nomes := make([]string, 0, len(aliases))
	for alias := range aliases {
		nomes = append(nomes, alias)
	}
	sort.Strings(nomes)

	partes := make([]string, 0, 12)
	for _, alias := range nomes {
		target := aliases[alias]
		if !visiveis[target] {
			continue
		}
		partes = append(partes, fmt.Sprintf("%s%s = %s%s", r.Prefix(), alias, r.Prefix(), target))
		if len(partes) >= 12 {
			break
		}
	}
	return strings.Join(partes, ", ")
}
