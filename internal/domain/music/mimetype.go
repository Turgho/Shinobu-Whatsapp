package music

// AudioMimetypeForExt mapeia a extensão do ficheiro (sem ponto) para o MIME usado no WhatsApp.
func AudioMimetypeForExt(ext string) string {
	switch ext {
	case "ogg", "opus":
		return "audio/ogg; codecs=opus"
	default:
		return "audio/mpeg"
	}
}
