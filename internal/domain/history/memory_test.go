package history

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatFacts_Empty(t *testing.T) {
	assert.Equal(t, "", FormatFacts(nil))
	assert.Equal(t, "", FormatFacts(map[string]string{}))
}

func TestFormatFacts_Single(t *testing.T) {
	facts := map[string]string{"nome": "João"}
	expected := "Fatos conhecidos sobre o usuário:\n- nome: João\n"
	assert.Equal(t, expected, FormatFacts(facts))
}

func TestFormatFacts_Multiple(t *testing.T) {
	facts := map[string]string{
		"nome":          "Maria",
		"jogo favorito": "Zelda",
		"idade":         "25",
	}
	result := FormatFacts(facts)
	assert.Contains(t, result, "- nome: Maria")
	assert.Contains(t, result, "- idade: 25")
	assert.Contains(t, result, "- jogo favorito: Zelda")
	assert.Contains(t, result, "Fatos conhecidos sobre o usuário:")
}

func TestFormatFacts_AlphabeticalOrder(t *testing.T) {
	facts := map[string]string{
		"z": "ultimo",
		"a": "primeiro",
		"m": "meio",
	}
	result := FormatFacts(facts)
	aIdx := strings.Index(result, "- a: primeiro")
	mIdx := strings.Index(result, "- m: meio")
	zIdx := strings.Index(result, "- z: ultimo")
	assert.True(t, aIdx < mIdx && mIdx < zIdx, "deve estar em ordem alfabética")
}

func TestExtractFactsFromPrompt_Nome(t *testing.T) {
	facts := ExtractFactsFromPrompt("Meu nome é João")
	assert.Equal(t, "joão", facts["nome"])

	facts = ExtractFactsFromPrompt("me chamo Maria")
	assert.Equal(t, "maria", facts["nome"])

	facts = ExtractFactsFromPrompt("Eu sou o Pedro")
	assert.Equal(t, "pedro", facts["nome"])

	facts = ExtractFactsFromPrompt("eu sou a Ana")
	assert.Equal(t, "ana", facts["nome"])
}

func TestExtractFactsFromPrompt_Idade(t *testing.T) {
	facts := ExtractFactsFromPrompt("minha idade é 25")
	assert.Equal(t, "25", facts["idade"])

	facts = ExtractFactsFromPrompt("tenho 30 anos")
	assert.Equal(t, "30", facts["idade"])
}

func TestExtractFactsFromPrompt_Jogo(t *testing.T) {
	facts := ExtractFactsFromPrompt("meu jogo favorito é Zelda")
	assert.Equal(t, "zelda", facts["jogo favorito"])
}

func TestExtractFactsFromPrompt_Hobby(t *testing.T) {
	facts := ExtractFactsFromPrompt("meu hobby é desenhar")
	assert.Equal(t, "desenhar", facts["hobby"])

	facts = ExtractFactsFromPrompt("meu hobbie é ler")
	assert.Equal(t, "ler", facts["hobby"])
}

func TestExtractFactsFromPrompt_Gosto(t *testing.T) {
	facts := ExtractFactsFromPrompt("gosto de pizza")
	assert.Equal(t, "pizza", facts["gosto"])

	facts = ExtractFactsFromPrompt("não gosto de café")
	assert.Equal(t, "café", facts["não gosto"])
}

func TestExtractFactsFromPrompt_Cidade(t *testing.T) {
	// Extrai só a primeira palavra porque IndexAny para no espaço
	facts := ExtractFactsFromPrompt("moro em São Paulo")
	assert.Equal(t, "são", facts["cidade"])

	facts = ExtractFactsFromPrompt("sou de Curitiba")
	assert.Equal(t, "curitiba", facts["cidade"])
}

func TestExtractFactsFromPrompt_Empty(t *testing.T) {
	assert.Equal(t, 0, len(ExtractFactsFromPrompt("")))
	assert.Equal(t, 0, len(ExtractFactsFromPrompt("qual é a previsão do tempo?")))
}

func TestExtractFactsFromPrompt_MaxLength(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyz0123456789" + "abcdefghijklmnopqrstuvwxyz0123456789"
	facts := ExtractFactsFromPrompt("meu nome é " + long)
	assert.NotContains(t, facts, "nome", "não deve extrair valores com >60 caracteres")
}
