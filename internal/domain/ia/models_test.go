package ia

import (
	"testing"
)

// A escolha do modelo precisa respeitar o papel de cada um:
// contínuo no dia a dia, elaborado em respostas longas e o override de
// config para testar personalidade.
func TestMainAnswerParams(t *testing.T) {
	// sem override e sem web → modelo contínuo
	p := mainAnswerParams(ModeBrief, false, "")
	if p.model != ModelContinuous {
		t.Errorf("esperava %s, veio %s", ModelContinuous, p.model)
	}

	// com busca web → modelo elaborado (contexto maior)
	p = mainAnswerParams(ModeBrief, true, "")
	if p.model != ModelElaborate {
		t.Errorf("com web esperava %s, veio %s", ModelElaborate, p.model)
	}

	// override da config (teste de personalidade) wins
	p = mainAnswerParams(ModeBrief, false, ModelPersonality)
	if p.model != ModelPersonality {
		t.Errorf("override ignorado: %s", p.model)
	}

	// no modo padrão (não brief) o override também vale
	p = mainAnswerParams(ModeNormal, false, ModelPersonality)
	if p.model != ModelPersonality {
		t.Errorf("override ignorado no modo padrão: %s", p.model)
	}
}

func TestConversationModel(t *testing.T) {
	var cfg *Config
	if got := cfg.ConversationModel(); got != ModelContinuous {
		t.Errorf("config nil deveria usar o padrão, veio %s", got)
	}

	cfg = &Config{Model: ModelPersonality}
	if got := cfg.ConversationModel(); got != ModelPersonality {
		t.Errorf("override não aplicado: %s", got)
	}
}

// Os três modelos são modelos de raciocínio: effort low (e hidden no qwen)
// é o que evita o "thinking mode" comendo o max_tokens.
func TestIsReasoningModel(t *testing.T) {
	casos := map[string]bool{
		ModelContinuous:        true,
		ModelPersonality:       true,
		ModelElaborate:         true,
		"llama-3.1-8b-instant": false,
	}
	for model, want := range casos {
		if got := isReasoningModel(model); got != want {
			t.Errorf("isReasoningModel(%q) = %v, esperava %v", model, got, want)
		}
	}
}
