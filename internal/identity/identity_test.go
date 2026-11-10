package identity

import "testing"

func TestOwnerMatchesPorJID(t *testing.T) {
	o := Owner{JID: "276807035179046@lid"}

	if !o.Matches("276807035179046", "276807035179046@lid") {
		t.Error("JID exato deveria casar")
	}
	if o.Matches("999999999999", "999999999999@lid") {
		t.Error("JID de outra pessoa não deveria casar")
	}
}

func TestOwnerMatchesPorNumero(t *testing.T) {
	o := Owner{Number: "276807035179046"}

	// remetente com endereço LID: User é o LID, JID traz o @lid
	if !o.Matches("276807035179046", "276807035179046@lid") {
		t.Error("número deveria casar pelo User")
	}
	if !o.Matches("outro", "276807035179046@lid") {
		t.Error("número deveria casar pelo JID")
	}
	if o.Matches("999999999999", "999999999999@lid") {
		t.Error("número de outra pessoa não deveria casar")
	}
}

func TestOwnerComFuncaoDoDispositivo(t *testing.T) {
	o := Owner{JID: "276807035179046@lid"}

	// ToNonAD remove o ":12", mas a comparação precisa ser tolerante
	if !o.Matches("276807035179046", "276807035179046:12@lid") {
		t.Error("JID com sufixo de dispositivo deveria casar")
	}
}

func TestOwnerVazioNuncaCasa(t *testing.T) {
	var o Owner
	if o.Matches("5511", "5511@s.whatsapp.net") {
		t.Error("owner sem configuração não deveria casar")
	}
}

func TestOwnerMatchesAnyComAdmins(t *testing.T) {
	o := Owner{Number: "111"}
	admins := []string{"222@s.whatsapp.net"}

	if !o.MatchesAny("111", "111@s.whatsapp.net", admins) {
		t.Error("dono deveria casar")
	}
	if !o.MatchesAny("222", "222@s.whatsapp.net", admins) {
		t.Error("admin por JID deveria casar")
	}
	if o.MatchesAny("333", "333@s.whatsapp.net", admins) {
		t.Error("terceiro não deveria casar")
	}
}
