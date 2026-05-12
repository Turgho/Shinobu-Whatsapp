package public

import (
	"context"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/music"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// PingCommand responde com latência estimada desde o timestamp da mensagem recebida.
func AudioEffectsCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	_ = utils.Reply(ctx, client, evt, "⏳ Modificando seu áudio...")

	audio, err := utils.DownloadFromEvent(ctx, client, evt, utils.FilterAudioOnly)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"📎 Envia um áudio e marque com o comando `!reverb_slowed`.")
	}

	effect, err := music.SlowedReverb(ctx, audio.Data, audio.Ext)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui converter a mídia. Certifique-se de que é audio ou música válido.")
	}

	uploaded, err := client.Upload(ctx, effect, whatsmeow.MediaAudio)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Falha ao enviar o audio modificado.")
	}

	if err := utils.SendAudio(ctx, client, evt, &uploaded, audio.Ext, false, true); err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Falha ao enviar a figurinha. Tente novamente.")
	}

	return nil
}
