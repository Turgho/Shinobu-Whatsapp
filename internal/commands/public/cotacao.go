package public

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/cotacao"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// formatoBR formata um float64 com duas casas decimais e vírgula como separador decimal.
func formatoBR(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	return strings.Replace(s, ".", ",", 1)
}

// CotacaoCommand busca e exibe cotações de USD e EUR em reais.
// Se args[0] for "dolar"/"usd"/"dólar"/"dollar", mostra só USD.
// Se args[0] for "euro"/"eur", mostra só EUR.
// Caso contrário, mostra ambas.
func CotacaoCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	cotacaoClient *cotacao.CotacaoClient,
) error {
	rates, err := cotacaoClient.Fetch(ctx)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgCotacaoFail)
	}

	filter := parseCurrencyFilter(args)
	if filter != "" {
		filtered := make([]cotacao.CotacaoResult, 0, 1)
		for _, r := range rates {
			if r.Code == filter {
				filtered = append(filtered, r)
			}
		}
		rates = filtered
	}

	if len(rates) == 0 {
		return whatsapp.Reply(ctx, client, evt, msgCotacaoFail)
	}

	// Garante ordem consistente: USD primeiro, depois EUR.
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Code < rates[j].Code
	})

	var b strings.Builder
	for _, r := range rates {
		var emoji string
		switch r.Code {
		case "USD":
			emoji = "💵"
		case "EUR":
			emoji = "💶"
		}

		sig := ""
		if r.PctChange >= 0 {
			sig = "+"
		}

		b.WriteString(fmt.Sprintf("%s *%s* R$ %s\n\n",
			emoji, r.Name, formatoBR(r.Bid)))
		b.WriteString(fmt.Sprintf("📈 Máx %s • Mín %s • Variação %s%s%%\n",
			formatoBR(r.High), formatoBR(r.Low), sig, formatoBR(r.PctChange)))
	}

	// A hora pode variar conforme o retorno da API.
	if len(rates) > 0 {
		timeStr := rates[0].UpdatedAt
		if idx := strings.LastIndex(timeStr, " "); idx != -1 {
			timeStr = timeStr[idx+1:]
		}
		b.WriteString(fmt.Sprintf("\n🕐 Atualizado às %s", timeStr))
	}

	return whatsapp.Reply(ctx, client, evt, b.String())
}

// parseCurrencyFilter interpreta o argumento textual para filtrar moeda.
// Retorna "USD", "EUR" ou vazio (mostrar todas).
func parseCurrencyFilter(args []string) string {
	if len(args) == 0 {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "dolar", "dólar", "usd", "dollar":
		return "USD"
	case "euro", "eur":
		return "EUR"
	default:
		return ""
	}
}

// CotacaoHandler cria um handler com as dependências injetadas.
func CotacaoHandler(cotacaoClient *cotacao.CotacaoClient) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client,
		evt *events.Message, args []string) error {
		return CotacaoCommand(ctx, client, evt, args, cotacaoClient)
	}
}
