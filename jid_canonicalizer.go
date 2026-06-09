package antiban

import "strings"

type JidCanonicalizer struct {
	resolver *LidResolver
}

func NewJidCanonicalizer(resolver *LidResolver) *JidCanonicalizer {
	return &JidCanonicalizer{resolver: resolver}
}

func (jc *JidCanonicalizer) CanonicalizeTarget(jid string) string {
	if jc == nil || jc.resolver == nil {
		return jid
	}
	return jc.resolver.ResolveCanonical(jid)
}

func (jc *JidCanonicalizer) CanonicalKey(jid string) string {
	canon := jc.CanonicalizeTarget(jid)
	canon = strings.TrimSuffix(canon, "@s.whatsapp.net")
	return canon
}

func (jc *JidCanonicalizer) OnIncomingEvent(fromJID, participant string) {
	if jc == nil || jc.resolver == nil {
		return
	}

	if strings.Contains(fromJID, "@lid") && participant != "" {
		lid := extractJIDUser(fromJID)
		pn := extractJIDUser(participant)
		if lid != "" && pn != "" && lid != pn {
			jc.resolver.Learn(lid, pn)
		}
	}
	if strings.Contains(participant, "@lid") && !strings.Contains(fromJID, "@lid") {
		lid := extractJIDUser(participant)
		pn := extractJIDUser(fromJID)
		if lid != "" && pn != "" && lid != pn {
			jc.resolver.Learn(lid, pn)
		}
	}
}

func extractJIDUser(jid string) string {
	at := strings.Index(jid, "@")
	if at == -1 {
		return jid
	}
	return jid[:at]
}
