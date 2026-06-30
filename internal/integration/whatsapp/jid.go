package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// StripLIDDeviceSuffix remove o sufixo :N de um JID @lid, retornando a
// forma base usada para buscar nome de contato no store do whatsmeow.
// Ex: "123456789:5@lid" → "123456789@lid"
// JIDs que não são @lid retornam inalterados.
func StripLIDDeviceSuffix(jid types.JID) types.JID {
	if jid.Server != types.HiddenUserServer {
		return jid
	}
	return types.NewJID(jid.User, jid.Server)
}

// ResolveContactName tenta resolver o nome de exibição de um contato,
// tentando a forma exata do JID primeiro e depois a forma base (sem
// sufixo de dispositivo) para JIDs @lid.
func ResolveContactName(client *whatsmeow.Client, jid types.JID) string {
	contact, err := client.Store.Contacts.GetContact(context.Background(), jid)
	if err == nil && contact.Found && contact.PushName != "" {
		return contact.PushName
	}

	if jid.Server == types.HiddenUserServer {
		baseJID := StripLIDDeviceSuffix(jid)
		if baseJID != jid {
			contact, err = client.Store.Contacts.GetContact(context.Background(), baseJID)
			if err == nil && contact.Found && contact.PushName != "" {
				return contact.PushName
			}
		}
	}

	return ""
}

// ResolvePNFromLID tenta obter o número de telefone (PN) correspondente
// a um JID @lid via o mapeamento interno do whatsmeow.
func ResolvePNFromLID(client *whatsmeow.Client, lidJID types.JID) (types.JID, bool) {
	pnJID, err := client.Store.LIDs.GetPNForLID(context.Background(), lidJID)
	if err != nil || pnJID.User == "" {
		return types.JID{}, false
	}
	return pnJID, true
}
