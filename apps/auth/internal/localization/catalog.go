package localization

import (
	"bytes"
	"errors"
	"strings"
	"text/template"
)

type Locale string

const (
	LocaleEnglish    Locale = "en"
	LocaleTurkish    Locale = "tr"
	LocaleFrench     Locale = "fr"
	LocalePortuguese Locale = "pt"
)

var ErrUnsupportedLocale = errors.New("unsupported identity locale")

type Message struct {
	Subject string
	Body    string
}

var catalog = map[Locale]map[string]Message{
	LocaleEnglish: {
		"email_verification": {Subject: "Verify your AviaSurveil360 email", Body: "Use this verification code: {{.Code}}"},
		"password_reset":     {Subject: "Reset your AviaSurveil360 password", Body: "Use this password reset code: {{.Code}}"},
		"mfa_recovery":       {Subject: "AviaSurveil360 MFA recovery", Body: "Use this recovery code: {{.Code}}"},
	},
	LocaleTurkish: {
		"email_verification": {Subject: "AviaSurveil360 e-postanızı doğrulayın", Body: "Bu doğrulama kodunu kullanın: {{.Code}}"},
		"password_reset":     {Subject: "AviaSurveil360 parolanızı sıfırlayın", Body: "Bu parola sıfırlama kodunu kullanın: {{.Code}}"},
		"mfa_recovery":       {Subject: "AviaSurveil360 MFA kurtarma", Body: "Bu kurtarma kodunu kullanın: {{.Code}}"},
	},
	LocaleFrench: {
		"email_verification": {Subject: "Vérifiez votre adresse e-mail AviaSurveil360", Body: "Utilisez ce code de vérification : {{.Code}}"},
		"password_reset":     {Subject: "Réinitialisez votre mot de passe AviaSurveil360", Body: "Utilisez ce code de réinitialisation : {{.Code}}"},
		"mfa_recovery":       {Subject: "Récupération MFA AviaSurveil360", Body: "Utilisez ce code de récupération : {{.Code}}"},
	},
	LocalePortuguese: {
		"email_verification": {Subject: "Verifique o seu e-mail AviaSurveil360", Body: "Use este código de verificação: {{.Code}}"},
		"password_reset":     {Subject: "Redefina a sua palavra-passe AviaSurveil360", Body: "Use este código de redefinição: {{.Code}}"},
		"mfa_recovery":       {Subject: "Recuperação MFA AviaSurveil360", Body: "Use este código de recuperação: {{.Code}}"},
	},
}

func Normalize(raw string) (Locale, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return LocaleEnglish, nil
	}
	base := strings.SplitN(value, "-", 2)[0]
	locale := Locale(base)
	if _, ok := catalog[locale]; !ok {
		return "", ErrUnsupportedLocale
	}
	return locale, nil
}

func MessageFor(locale Locale, key string) (Message, error) {
	normalized, err := Normalize(string(locale))
	if err != nil {
		return Message{}, err
	}
	message, ok := catalog[normalized][key]
	if !ok {
		return Message{}, errors.New("identity message key is missing")
	}
	return message, nil
}

func Locales() []Locale {
	return []Locale{LocaleEnglish, LocaleTurkish, LocaleFrench, LocalePortuguese}
}

func Keys() []string {
	return []string{"email_verification", "password_reset", "mfa_recovery"}
}

func Render(locale Locale, key string, data any) (Message, error) {
	message, err := MessageFor(locale, key)
	if err != nil {
		return Message{}, err
	}
	templateValue, err := template.New(key).Option("missingkey=error").Parse(message.Body)
	if err != nil {
		return Message{}, err
	}
	var body bytes.Buffer
	if err := templateValue.Execute(&body, data); err != nil {
		return Message{}, err
	}
	message.Body = body.String()
	return message, nil
}
