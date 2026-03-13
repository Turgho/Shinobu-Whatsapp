package bot

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Turgho/YuukoWhatsapp/pkg/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"

	qrterminal "github.com/mdp/qrterminal/v3"
)

type Client struct {
	WAClient *whatsmeow.Client
}

// NewClient cria um novo client do WhatsApp usando SQLite
func NewClient(ctx context.Context, db *sql.DB) (*Client, error) {
	// Cria um logger para o banco de dados
	dbLogger := logger.NewDatabaseLogger()

	// Configura o armazenamento do dispositivo usando o banco de dados
	container := sqlstore.NewWithDB(db, "sqlite3", dbLogger)

	// Cria as tabelas se não existirem
	if err := container.Upgrade(ctx); err != nil {
		return nil, err
	}

	// Tenta pegar o primeiro device do banco
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		// Nenhum device existe ainda, cria um novo
		deviceStore = container.NewDevice()

		fmt.Println("Novo dispositivo criado. Escaneie o QR Code para autenticar!")
	}

	// Cria um logger para o cliente WhatsApp
	waLogger := logger.NewWhatsAppLogger()

	// Cria o client WhatsApp com o deviceStore
	client := whatsmeow.NewClient(deviceStore, waLogger)

	return &Client{
		WAClient: client,
	}, nil
}

// Connect conecta o client ao WhatsApp e gera QR code se necessário
func (c *Client) Connect(ctx context.Context) error {
	if c.WAClient.Store.ID == nil {
		// Gera um novo QR code para login caso não haja um ID armazenado
		qrChan, _ := c.WAClient.GetQRChannel(ctx)

		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					fmt.Println("Escaneie o QR Code:")
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				}
			}
		}()
	}

	// Conecta ao WhatsApp
	err := c.WAClient.Connect()
	if err != nil {
		return err
	}

	return nil
}

// RegisterHandlers permite registrar um manipulador de eventos para o cliente WhatsApp
func (c *Client) RegisterHandlers(handler func(evt interface{})) {
	c.WAClient.AddEventHandler(handler)
}

// Listen inicia o loop de escuta e aguarda sinais de interrupção
func (c *Client) Listen() {
	fmt.Println("Bot rodando. Pressione CTRL+C para parar.")

	// Cria um canal para capturar sinais do sistema
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Bloqueia até receber um sinal
	sig := <-sigs
	fmt.Println("\nRecebido sinal:", sig)
	fmt.Println("Encerrando bot...")
	c.WAClient.Disconnect() // desconecta do WhatsApp
}
