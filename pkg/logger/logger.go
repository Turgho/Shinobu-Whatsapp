package logger

import waLog "go.mau.fi/whatsmeow/util/log"

// NewWhatsAppLogger cria um logger para o cliente WhatsApp.
// Nível WARN: só aparecem avisos e erros, mensagens recebidas não são logadas.
func NewWhatsAppLogger() waLog.Logger {
	return waLog.Stdout("WhatsApp", "WARN", true)
}

// NewDatabaseLogger cria um logger para o banco de dados.
// Nível WARN: só aparecem problemas reais de banco.
func NewDatabaseLogger() waLog.Logger {
	return waLog.Stdout("Database", "WARN", true)
}
