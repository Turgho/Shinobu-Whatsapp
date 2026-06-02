package gosafe

import (
	"log"
)

func Go(fn func()) {
	go func() {
		// Recupera qualquer panic na goroutine, loga e deixa a aplicação continuar.
		// Sem isso, um panic numa goroutine filha derruba o processo inteiro.
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[gosafe] panic recuperado: %v", r)
			}
		}()
		fn()
	}()
}
