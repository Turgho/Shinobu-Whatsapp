package public

// Mensagens amigáveis para erros comuns — voz da Shinobu, sem expor detalhes internos.
const (
	// Clima
	msgCityNotFound    = "Não achei esse lugar. Pode repetir o nome?"
	msgWeatherFailed   = "Não consegui buscar o clima agora. Tenta de novo em alguns segundos."
	msgNeedCityName    = "Fala o nome da cidade também."

	// Agenda
	msgInvalidDate         = "Não entendi a data. Tenta assim: amanhã às 9h, ou 28/06 14:00."
	msgPastDate            = "Não posso agendar lembretes no passado, tolo."
	msgDateLimit           = "Só posso agendar com até 30 dias de antecedência."
	msgSaveReminderFail    = "Não consegui salvar o lembrete. Tenta de novo."
	msgRemoveReminderFail  = "Não consegui remover o lembrete."
	msgInvalidNumber       = "Esse número não é válido."
	msgNoReminders         = "Nenhum lembrete agendado."
	msgReminderUsage       = "Use: `agenda remover <número>`"

	// Play
	msgNoQuery        = "Fala o nome ou link da música."
	msgDownloadFail    = "Não achei essa música. Tenta com outro nome."
	msgSendAudioFail   = "Não consegui enviar o áudio."

	// Efeito
	msgNoAudioForEffect = "Manda um áudio primeiro, ou responde a um áudio com o comando."
	msgProcessAudioFail = "Não consegui processar esse áudio. Manda outro."
	msgUploadAudioFail  = "Não consegui enviar o áudio modificado."

	// Sticker
	msgNoMediaForSticker = "Manda uma imagem ou vídeo, ou responde a uma mídia com o comando."
	msgConvertStickerFail = "Não consegui criar a figurinha. Tem certeza que é uma imagem ou vídeo?"
	msgSendStickerFail    = "Não consegui enviar a figurinha."

	// Shinobu
	msgShinobuFail = "Algo deu errado. Fala de novo?"

	// Cotação
	msgCotacaoFail = "Não consegui buscar as cotações agora. Tenta de novo em alguns segundos."

	// Feriado
	msgFeriadoFail = "Não consegui buscar os feriados agora. Tenta de novo em alguns segundos."
	msgFeriadoNone = "Não encontrei feriados próximos."

	// Notícia
	msgNoticiaFail = "Não consegui buscar as notícias agora. Tenta de novo."

	// Receita
	msgReceitaNoQuery = "Qual receita você quer? Ex: !receita bolo de cenoura"
	msgReceitaFail    = "Não achei receita disso. Tenta outro prato."

	// Piada
	msgPiadaFail = "Não consegui pensar em uma piada agora. Tenta de novo."

	// Fato
	msgFatoFail = "Não consegui lembrar de um fato agora. Tenta de novo."

	// Filme
	msgFilmeFail = "Não consegui recomendar um filme agora. Tenta de novo."

	// Contagem
	msgContagemUsage = "Usa assim: !contagem natal 25/12"
	msgContagemPast  = "Essa data já passou!"

	// Unsticker
	msgUnstickerNoMedia = "Cite ou envie uma figurinha para converter."
	msgUnstickerFail    = "Não consegui converter essa figurinha."
	msgUnstickerCaption = "📎 Aqui está a mídia original."

	// Traduz
	msgTraduzNoText = "Cite uma mensagem ou escreva o texto após !traduz."
	msgTraduzFail   = "Não consegui traduzir agora. Tenta de novo."
)
