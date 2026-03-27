package domain

import "testing"

func TestIsBot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeName string
		login    string
		want     bool
	}{
		// Tier 1: __typename == "Bot" (authoritative, any login)
		{"typename bot with codex login", "Bot", "chatgpt-codex-connector", true},
		{"typename bot with coderabbit login", "Bot", "coderabbitai", true},
		{"typename bot with cursor login", "Bot", "cursor", true},
		{"typename bot with copilot login", "Bot", "copilot-pull-request-reviewer", true},
		{"typename bot with unknown login", "Bot", "some-new-bot-nobody-knows", true},
		{"typename bot empty login", "Bot", "", true},

		// Tier 2: [bot] suffix (REST API format)
		{"codex rest format", "", "chatgpt-codex-connector[bot]", true},
		{"coderabbit rest format", "", "coderabbitai[bot]", true},
		{"cursor rest format", "", "cursor[bot]", true},
		{"copilot rest format", "", "copilot-pull-request-reviewer[bot]", true},
		{"sourcery rest format", "", "sourcery-ai[bot]", true},
		{"dependabot rest format", "", "dependabot[bot]", true},
		{"unknown bot rest format", "", "custom-enterprise-bot[bot]", true},
		{"typename user but bot suffix", "User", "weird[bot]", true},

		// Tier 3: knownBots fallback map (GraphQL logins without suffix)
		{"codex graphql login", "", "chatgpt-codex-connector", true},
		{"coderabbit graphql login", "", "coderabbitai", true},
		{"copilot reviewer graphql", "", "copilot-pull-request-reviewer", true},
		{"copilot agent graphql", "", "copilot", true},
		{"sourcery graphql", "", "sourcery-ai", true},
		{"codacy graphql", "", "codacy-production", true},
		{"sonarcloud graphql", "", "sonarcloud", true},
		{"sonarqubecloud graphql", "", "sonarqubecloud", true},
		{"sonarqube-cloud-us graphql", "", "sonarqube-cloud-us", true},
		{"dependabot graphql", "", "dependabot", true},
		{"renovate graphql", "", "renovate", true},
		{"github-actions graphql", "", "github-actions", true},

		// False positives that must NOT match
		{"cursor is ambiguous without typename", "", "cursor", false},
		{"human user", "User", "alice", false},
		{"human user with bot in name", "User", "robotfan", false},
		{"human user cursor prefix", "User", "cursor-fan", false},
		{"human user mybot", "", "mybot", false},
		{"human user renovate-helper", "", "renovate-helper", false},
		{"human user with org typename", "Organization", "some-org", false},
		{"empty everything", "", "", false},
		{"typename user empty login", "User", "", false},
		{"typename mannequin", "Mannequin", "ghost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsBot(tt.typeName, tt.login)
			if got != tt.want {
				t.Errorf("IsBot(%q, %q) = %v, want %v", tt.typeName, tt.login, got, tt.want)
			}
		})
	}
}
