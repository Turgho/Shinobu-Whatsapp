package logger

import waLog "go.mau.fi/whatsmeow/util/log"

// NewWhatsAppLogger cria um logger para o cliente WhatsApp
func NewWhatsAppLogger() waLog.Logger {
	return waLog.Stdout("WhatsApp", "INFO", true)
}

// NewDatabaseLogger cria um logger para o banco de dados
func NewDatabaseLogger() waLog.Logger {
	return waLog.Stdout("Database", "INFO", true)
}
