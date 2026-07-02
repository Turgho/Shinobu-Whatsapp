package ia

import (
	"fmt"
	"strings"
	"sync"
)

// nluCommands é a única fonte de verdade para comandos reconhecidos pelo NLU.
// Usada tanto para gerar o prompt dinamicamente quanto para validar
// a saída do modelo via DispatchableCommand.
var nluCommands = []string{
	"clima", "play", "sticker", "efeito", "aniversário",
	"agenda", "cotacao", "feriado", "noticia", "receita",
	"piada", "fato", "filme", "contagem", "unsticker", "traduz",
}

// nluCommandDesc alimenta a seção "COMANDOS:" do prompt.
// Mantida separada de CommandMeta em app_commands.go porque descreve
// o formato de args esperado pelo NLU, não a descrição para usuarios.
var nluCommandDesc = map[string]string{
	"clima":       "previsão do tempo. Args: [cidade] ou [cidade, YYYY-MM-DD]",
	"play":        "tocar música. Args: [nome/banda]",
	"sticker":     "criar figurinha. Args: []",
	"efeito":      "efeito de áudio. Args: [tipo, intensidade?]",
	"aniversário": "aniversários do grupo. Args: [lista]",
	"agenda":      "lembrete/agendamento. Args: [data+hora, mensagem]",
	"cotacao":     "cotação de moeda (dólar/euro). Args: [moeda]",
	"feriado":     "próximos feriados. Args: [N]",
	"noticia":     "últimas notícias. Args: []",
	"receita":     "receita culinária. Args: [prato]",
	"piada":       "piada engraçada. Args: []",
	"fato":        "fato curioso. Args: []",
	"filme":       "recomendação de filme. Args: [gênero]",
	"contagem":    "contagem regressiva de dias. Args: [evento, DD/MM]",
	"unsticker":   "converter figurinha em imagem. Args: []",
	"traduz":      "tradução de texto. Args: [instrução]",
}

var (
	nluCommandSet  map[string]struct{}
	nluCommandOnce sync.Once
)

func initNLUSet() {
	nluCommandSet = make(map[string]struct{}, len(nluCommands)+1)
	for _, n := range nluCommands {
		nluCommandSet[n] = struct{}{}
	}
	// Variante sem acento que o modelo pode eventualmente produzir
	nluCommandSet["aniversario"] = struct{}{}
}

// DispatchableCommand checa se o nome está na whitelist de comandos públicos do NLU.
// Substitui o switch anterior por lookup O(1) em map.
func DispatchableCommand(name string) bool {
	nluCommandOnce.Do(initNLUSet)
	_, ok := nluCommandSet[name]
	return ok
}

// buildNLUPromptSection gera a seção "COMANDOS:" do prompt a partir da
// lista única nluCommands + nluCommandDesc.
func buildNLUPromptSection() string {
	var b strings.Builder
	b.WriteString("COMANDOS:\n")
	for _, name := range nluCommands {
		desc := nluCommandDesc[name]
		b.WriteString(fmt.Sprintf("- %s: %s\n", name, desc))
	}
	return b.String()
}
