package domain

import "strings"

// knownBots is a fallback set for logins that may not have __typename available.
// Only includes unambiguous bot logins verified against live GitHub API responses.
// Primary detection is via GraphQL __typename == "Bot".
var knownBots = map[string]struct{}{
	"chatgpt-codex-connector":       {}, // OpenAI Codex
	"coderabbitai":                  {}, // CodeRabbit
	"copilot-pull-request-reviewer": {}, // GitHub Copilot code review
	"copilot":                       {}, // Copilot coding agent
	"sourcery-ai":                   {}, // Sourcery
	"codacy-production":             {}, // Codacy
	"sonarcloud":                    {}, // SonarCloud
	"sonarqubecloud":                {}, // SonarQube Cloud
	"sonarqube-cloud-us":            {}, // SonarQube Cloud US
	"dependabot":                    {}, // Dependabot
	"renovate":                      {}, // Renovate
	"github-actions":                {}, // GitHub Actions
}

// IsBot reports whether the author is a bot account.
//
// Detection is three-tier:
//  1. typeName == "Bot" — authoritative from GitHub GraphQL __typename
//  2. login ends with "[bot]" — standard GitHub App suffix in REST API
//  3. login is in the knownBots fallback map
//
// The typeName parameter is the GraphQL __typename field ("Bot", "User", "Organization", etc.).
// Pass an empty string when __typename is unavailable.
func IsBot(typeName, login string) bool {
	if typeName == "Bot" {
		return true
	}
	if strings.HasSuffix(login, "[bot]") {
		return true
	}
	_, ok := knownBots[login]
	return ok
}
