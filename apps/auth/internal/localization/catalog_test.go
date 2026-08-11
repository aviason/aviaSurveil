package localization

import (
	"strings"
	"testing"
)

func TestAllRequiredLocalesAndMessageKeysRender(t *testing.T) {
	for _, locale := range Locales() {
		for _, key := range Keys() {
			message, err := Render(locale, key, struct{ Code string }{Code: "CODE-123"})
			if err != nil {
				t.Fatalf("render %s/%s: %v", locale, key, err)
			}
			if message.Subject == "" || !strings.Contains(message.Body, "CODE-123") {
				t.Fatalf("rendered %s/%s = %+v", locale, key, message)
			}
		}
	}
}

func TestLocaleNormalizationAndUnknownLocaleDenial(t *testing.T) {
	for _, input := range []string{"", "en-US", "tr-TR", "fr-FR", "pt-BR", "pt-PT"} {
		if _, err := Normalize(input); err != nil {
			t.Fatalf("normalize %q: %v", input, err)
		}
	}
	if _, err := Normalize("de-DE"); err == nil {
		t.Fatal("unknown locale accepted")
	}
}

func TestTemplateMissingDataFailsClosed(t *testing.T) {
	if _, err := Render(LocaleEnglish, "email_verification", struct{}{}); err == nil {
		t.Fatal("missing template data accepted")
	}
}
