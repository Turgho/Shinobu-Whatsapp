package public

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/integration/media"
	"github.com/Turgho/YuukoWhatsapp/internal/integration/whatsapp"
	"github.com/Turgho/YuukoWhatsapp/internal/domain/music"
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
		return whatsapp.SendText(ctx, client, evt, music.EffectList(), true)
	}

	effectName := strings.ToLower(args[0])

	if _, ok := music.Effects[effectName]; !ok {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Efeito não encontrado.\n\n"+music.EffectList(), true)
	}

	// Segundo arg opcional: intensidade (leve, medio, forte)
	intensity := music.IntensityMedium
	if len(args) >= 2 {
		intensity = music.ParseIntensity(strings.ToLower(args[1]))
	}

	intensityLabel := music.IntensityLabel(intensity)
	_ = whatsapp.SendText(ctx, client, evt,
		fmt.Sprintf("⏳ Aplicando *%s* (%s)...", effectName, intensityLabel), true)

	dl, err := media.DownloadFromEvent(ctx, client, evt, media.FilterAudioOnly)
	if err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"📎 Envie um áudio e chame `!efeito <nome>`, ou responda a um áudio com o comando.", true)
	}

	result, err := music.Apply(ctx, dl.Data, dl.Ext, effectName, intensity)
	if err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Não consegui processar o áudio. Certifique-se de que é um áudio válido.", true)
	}

	uploaded, err := client.Upload(ctx, result, whatsmeow.MediaAudio)
	if err != nil {
		return whatsapp.SendText(ctx, client, evt, "❌ Falha ao enviar o áudio modificado.", true)
	}

	if err := whatsapp.SendAudio(ctx, client, evt, &uploaded, "audio/mpeg", false, true); err != nil {
		return whatsapp.SendText(ctx, client, evt, "❌ Falha ao enviar o áudio.", true)
	}

	return nil
}
