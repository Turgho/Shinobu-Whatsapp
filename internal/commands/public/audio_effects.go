package public

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/music"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// AudioEffectsCommand aplica um efeito de áudio em um áudio enviado ou citado.
//
// Uso:
//
//	!efeito                  — lista os efeitos disponíveis
//	!efeito reverb           — intensidade média (padrão)
//	!efeito reverb leve      — intensidade leve
//	!efeito reverb forte     — intensidade forte
func AudioEffectsCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	if len(args) == 0 || strings.ToLower(args[0]) == "lista" {
		return utils.SendText(ctx, client, evt, music.EffectList(), true)
	}

	effectName := strings.ToLower(args[0])

	if _, ok := music.Effects[effectName]; !ok {
		return utils.SendText(ctx, client, evt,
			"❌ Efeito não encontrado.\n\n"+music.EffectList(), true)
	}

	// Segundo arg opcional: intensidade (leve, medio, forte)
	intensity := music.IntensityMedium
	if len(args) >= 2 {
		intensity = music.ParseIntensity(strings.ToLower(args[1]))
	}

	intensityLabel := []string{"leve", "médio", "forte"}[intensity]
	_ = utils.SendText(ctx, client, evt,
		fmt.Sprintf("⏳ Aplicando *%s* (%s)...", effectName, intensityLabel), true)

	audio, err := utils.DownloadFromEvent(ctx, client, evt, utils.FilterAudioOnly)
	if err != nil {
		return utils.SendText(ctx, client, evt,
			"📎 Envie um áudio e chame `!efeito <nome>`, ou responda a um áudio com o comando.", true)
	}

	result, err := music.Apply(ctx, audio.Data, audio.Ext, effectName, intensity)
	if err != nil {
		return utils.SendText(ctx, client, evt,
			"❌ Não consegui processar o áudio. Certifique-se de que é um áudio válido.", true)
	}

	uploaded, err := client.Upload(ctx, result, whatsmeow.MediaAudio)
	if err != nil {
		return utils.SendText(ctx, client, evt, "❌ Falha ao enviar o áudio modificado.", true)
	}

	if err := utils.SendAudio(ctx, client, evt, &uploaded, "audio/mpeg", false, true); err != nil {
		return utils.SendText(ctx, client, evt, "❌ Falha ao enviar o áudio.", true)
	}

	return nil
}
