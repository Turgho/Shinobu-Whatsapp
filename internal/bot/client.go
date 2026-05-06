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

// NewClient cria o client WhatsApp usando o banco de dados passado.
// Tenta reutilizar um dispositivo existente; cria um novo apenas se não houver nenhum.
func NewClient(ctx context.Context, db *sql.DB) (*Client, error) {
	dbLogger := logger.NewDatabaseLogger()
	container := sqlstore.NewWithDB(db, "sqlite3", dbLogger)

	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("falha ao atualizar schema do whatsmeow: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		// Erro real de banco — não silenciar nem criar device em cima de uma falha
		return nil, fmt.Errorf("erro ao buscar dispositivo no banco: %w", err)
	}

	// GetFirstDevice retorna nil quando não há nenhum dispositivo cadastrado
	if deviceStore == nil {
		deviceStore = container.NewDevice()
		fmt.Println("Nenhum dispositivo encontrado. Escaneie o QR Code para autenticar.")
	}

	waLogger := logger.NewWhatsAppLogger()
	client := whatsmeow.NewClient(deviceStore, waLogger)

	return &Client{WAClient: client}, nil
}

// Connect conecta ao WhatsApp. Se não houver sessão salva, exibe o QR Code.
func (c *Client) Connect(ctx context.Context) error {
	if c.WAClient.Store.ID == nil {
		qrChan, err := c.WAClient.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("erro ao obter canal de QR Code: %w", err)
		}

		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					fmt.Println("Escaneie o QR Code:")
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				}
			}
		}()
	}

	if err := c.WAClient.Connect(); err != nil {
		return fmt.Errorf("erro ao conectar no WhatsApp: %w", err)
	}

	return nil
}

// RegisterHandlers registra um handler de eventos no cliente WhatsApp.
func (c *Client) RegisterHandlers(handler func(evt interface{})) {
	c.WAClient.AddEventHandler(handler)
}

// Listen bloqueia até receber SIGINT ou SIGTERM e desconecta o bot graciosamente.
func (c *Client) Listen() {
	fmt.Println("Bot rodando. Pressione CTRL+C para parar.")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs
	fmt.Println("\nRecebido sinal:", sig)
	fmt.Println("Encerrando bot...")
	c.WAClient.Disconnect()
}
