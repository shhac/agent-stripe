package auth

import (
	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
)

func addCredentialType(item map[string]any, profile config.Profile) {
	item["credential_type"] = profileCredentialType(profile)
}

func profileCredentialType(profile config.Profile) string {
	if profile.CredentialType == "" {
		return credential.UnknownType
	}
	return profile.CredentialType
}

func addCredentialTypeHint(item map[string]any, credentialType string, missing bool, alias string) {
	switch {
	case missing:
		item["hint"] = "Credential type is not stored for this profile yet. Run 'agent-stripe auth check " + alias + "' to refresh profile metadata."
	case credentialType == credential.UnknownType:
		item["hint"] = "Stored credential format is not recognized by agent-stripe. It may still work; run 'agent-stripe auth check " + alias + "' to test it."
	case credential.IsPublishableType(credentialType):
		item["hint"] = "Publishable keys cannot authenticate agent-stripe API requests. Run 'agent-stripe auth update " + alias + " --form' with a restricted or secret key."
	}
}

func refreshStoredCredentialType(resolved *shared.ResolvedProfile, credentialType string) string {
	if resolved.CredentialSource != "keychain" || resolved.Alias == "" {
		return "not_stored"
	}
	profile, ok := config.Read().Profiles[resolved.Alias]
	if !ok {
		return "not_stored"
	}
	if profile.CredentialType == credentialType {
		return "current"
	}
	if err := config.UpdateProfile(resolved.Alias, func(profile config.Profile) config.Profile {
		profile.CredentialType = credentialType
		return profile
	}); err != nil {
		return "refresh_failed"
	}
	return "updated"
}
