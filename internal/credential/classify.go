package credential

import "strings"

const UnknownType = "unknown"

const (
	RestrictedLiveType  = "rk_live"
	RestrictedTestType  = "rk_test"
	SecretLiveType      = "sk_live"
	SecretTestType      = "sk_test"
	PublishableLiveType = "pk_live"
	PublishableTestType = "pk_test"
)

var knownTypes = []string{
	RestrictedLiveType,
	RestrictedTestType,
	SecretLiveType,
	SecretTestType,
	PublishableLiveType,
	PublishableTestType,
}

func Type(apiKey string) string {
	for _, credentialType := range knownTypes {
		if strings.HasPrefix(apiKey, credentialType+"_") {
			return credentialType
		}
	}
	return UnknownType
}

func IsPublishableType(credentialType string) bool {
	return credentialType == PublishableLiveType || credentialType == PublishableTestType
}
