package main

import (
	"log"

	"github.com/Turgho/Shinobu-Whatsapp/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
