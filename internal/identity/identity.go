// Package identity centraliza a regra "quem é o dono" do bot.
//
// O projeto usava três comparações diferentes para a mesma pergunta
// (JID completo, só dígitos e string crua) e duas configurações distintas
// (usersJID.owner e owner.number). Com o endereçamento LID da conta, o
// "Sender.User" é o LID — e as comparações divergiam: o dono podia ficar
// sem acesso aos comandos privados.
//
// A regra aqui aceita as duas formas (número e JID), sem depender do modo
// de endereçamento ativo.
package identity

import (
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/phone"
)

// Owner identifica o dono do bot. Number e JID são independentes: basta um
// deles estar configurado.
type Owner struct {
	// Number é o número do dono, só dígitos (ex.: 5511999999999).
	Number string
	// JID é o JID completo do dono (ex.: 5511999999999@lid).
	JID string
}

// Matches informa se o remetente é o dono.
//
// senderUser é o identificador do remetente sem sufixo (número ou LID) e
// senderJID é o JID completo (ToNonAD). Qualquer um dos dois casar com a
// configuração já vale.
func (o Owner) Matches(senderUser, senderJID string) bool {
	if o.JID != "" && senderJID != "" && normalizeJID(o.JID) == normalizeJID(senderJID) {
		return true
	}

	want := phone.NormalizeNumber(o.Number)
	if want == "" {
		return false
	}

	// número puro do remetente (sem sufixo @lid/@s.whatsapp.net)
	if got := phone.NormalizeNumber(senderUser); got != "" && got == want {
		return true
	}

	// número dentro do JID do remetente ( LID ou @s.whatsapp.net )
	if got := phone.NormalizeNumber(stripServer(senderJID)); got != "" && got == want {
		return true
	}

	return false
}

// MatchesAny informa se o remetente casa com o dono ou com um dos admins.
// Os admins são configurados como JID, mas também aceitam número.
func (o Owner) MatchesAny(senderUser, senderJID string, admins []string) bool {
	if o.Matches(senderUser, senderJID) {
		return true
	}
	for _, admin := range admins {
		adminOwner := Owner{Number: admin, JID: admin}
		if adminOwner.Matches(senderUser, senderJID) {
			return true
		}
	}
	return false
}

// normalizeJID padroniza um JID para comparação (minúsculas, sem o
// sufixo de dispositivo ":12").
func normalizeJID(jid string) string {
	jid = stripDevice(jid)
	return lower(jid)
}

// stripDevice remove o sufixo de dispositivo do JID
// (ex.: "5511:12@s.whatsapp.net" → "5511@s.whatsapp.net"), preservando
// o servidor (@lid / @s.whatsapp.net).
func stripDevice(jid string) string {
	at := strings.IndexByte(jid, '@')
	if at < 0 {
		return jid
	}
	user := jid[:at]
	if colon := strings.IndexByte(user, ':'); colon >= 0 {
		user = user[:colon]
	}
	return user + jid[at:]
}

// stripServer remove o sufixo do servidor (ex.: "5511@lid" → "5511").
func stripServer(jid string) string {
	for i := 0; i < len(jid); i++ {
		if jid[i] == '@' {
			return jid[:i]
		}
	}
	return jid
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
