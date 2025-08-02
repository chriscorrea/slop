package app

import (
	"io"
	"log/slog"
	"testing"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestDetermineSentimentExitCode(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected int
	}{
		{
			name:     "Positive at start",
			response: "PoSiTiVe - this is an example of great news",
			expected: ExitPositive,
		},
		{
			name:     "Negative at start",
			response: "NEGATIVE - this is bad",
			expected: ExitNegative,
		},
		{
			name:     "Neutral at start",
			response: "neutral - neither good nor bad",
			expected: ExitNeutral,
		},
		{
			name:     "Positive later in response",
			response: "The result is positive, subsequent text goes here",
			expected: ExitPositive,
		},
		{
			name:     "Negative later in response",
			response: "The result is negative, subsequent text goes here",
			expected: ExitNegative,
		},
		{
			name:     "No match",
			response: "The carrot cake is edible",
			expected: 0,
		},
		// multilingual tests - Spanish
		{
			name:     "Spanish positive",
			response: "bueno, el resultado es excelente",
			expected: ExitPositive,
		},
		{
			name:     "Spanish negative",
			response: "malo, esto es terrible",
			expected: ExitNegative,
		},
		{
			name:     "Spanish neutral",
			response: "neutro, es regular",
			expected: ExitNeutral,
		},
		// multilingual tests - French
		{
			name:     "French positive",
			response: "bon travail, c'est excellent",
			expected: ExitPositive,
		},
		{
			name:     "French negative",
			response: "mauvais résultat, c'est horrible",
			expected: ExitNegative,
		},
		{
			name:     "French neutral",
			response: "neutre, c'est moyen",
			expected: ExitNeutral,
		},
		// multilingual tests - German
		{
			name:     "German positive",
			response: "gut gemacht, ausgezeichnet!",
			expected: ExitPositive,
		},
		{
			name:     "German negative",
			response: "schlecht, das ist furchtbar",
			expected: ExitNegative,
		},
		{
			name:     "German neutral",
			response: "neutral, durchschnittlich",
			expected: ExitNeutral,
		},
		// multilingual tests - Portuguese
		{
			name:     "Portuguese positive",
			response: "bom trabalho, fantástico!",
			expected: ExitPositive,
		},
		{
			name:     "Portuguese negative",
			response: "mau resultado, terrível",
			expected: ExitNegative,
		},
		// multilingual tests - Italian
		{
			name:     "Italian positive",
			response: "buono, eccellente lavoro",
			expected: ExitPositive,
		},
		{
			name:     "Italian negative",
			response: "cattivo, è orribile",
			expected: ExitNegative,
		},
		// multilingual tests - Dutch
		{
			name:     "Dutch positive",
			response: "goed gedaan, fantastisch!",
			expected: ExitPositive,
		},
		{
			name:     "Dutch negative",
			response: "slecht, verschrikkelijk",
			expected: ExitNegative,
		},
		// multilingual tests - Nordic
		{
			name:     "Swedish positive",
			response: "bra jobbat, utmärkt!",
			expected: ExitPositive,
		},
		{
			name:     "Nordic negative",
			response: "dålig, fruktansvärd",
			expected: ExitNegative,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineSentimentExitCode(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeterminePassFailExitCode(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected int
	}{
		{
			name:     "Pass at start",
			response: "PASS - all tests successful",
			expected: ExitPass,
		},
		{
			name:     "Fail at start",
			response: "FAIL - tests failed",
			expected: ExitFail,
		},
		{
			name:     "Success synonym",
			response: "SUCCESS - operation completed",
			expected: ExitPass,
		},
		{
			name:     "Error synonym",
			response: "ERROR - something went wrong",
			expected: ExitFail,
		},
		{
			name:     "No match",
			response: "The process is running",
			expected: 0,
		},
		{
			name:     "Case insensitive",
			response: "failed to connect",
			expected: ExitFail,
		},
		// multilingual tests - Spanish
		{
			name:     "Spanish pass",
			response: "aprobado - todo correcto",
			expected: ExitPass,
		},
		{
			name:     "Spanish fail",
			response: "rechazado - hay errores",
			expected: ExitFail,
		},
		// multilingual tests - French
		{
			name:     "French pass",
			response: "réussi - tout est correct",
			expected: ExitPass,
		},
		{
			name:     "French fail",
			response: "échoué - il y a des erreurs",
			expected: ExitFail,
		},
		// multilingual tests - German
		{
			name:     "German pass",
			response: "bestanden - alles korrekt",
			expected: ExitPass,
		},
		{
			name:     "German fail",
			response: "fehlgeschlagen - es gibt Fehler",
			expected: ExitFail,
		},
		// multilingual tests - Portuguese
		{
			name:     "Portuguese pass",
			response: "aprovado com sucesso",
			expected: ExitPass,
		},
		{
			name:     "Portuguese fail",
			response: "falhou - erro encontrado",
			expected: ExitFail,
		},
		// multilingual tests - Italian
		{
			name:     "Italian pass",
			response: "approvato - tutto corretto",
			expected: ExitPass,
		},
		{
			name:     "Italian fail",
			response: "fallito - errore trovato",
			expected: ExitFail,
		},
		// multilingual tests - Dutch
		{
			name:     "Dutch pass",
			response: "geslaagd - alles correct",
			expected: ExitPass,
		},
		{
			name:     "Dutch fail",
			response: "mislukt - fout gevonden",
			expected: ExitFail,
		},
		// multilingual tests - Nordic
		{
			name:     "Swedish pass",
			response: "godkänd - allt korrekt",
			expected: ExitPass,
		},
		{
			name:     "Nordic fail",
			response: "misslyckades - fel hittades",
			expected: ExitFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determinePassFailExitCode(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnhanceSystemPromptForExitCode(t *testing.T) {
	tests := []struct {
		name       string
		basePrompt string
		exitMode   string
		expected   string
	}{
		{
			name:       "Sentiment mode with base prompt",
			basePrompt: "You are a helpful baker.",
			exitMode:   "sentiment",
			expected:   "You are a helpful baker.\n\nIMPORTANT: Start your response with exactly one of these words: POSITIVE, NEGATIVE, or NEUTRAL. Be direct and clear.",
		},
		{
			name:       "Pass-fail mode with base prompt",
			basePrompt: "You are a helpful baker.",
			exitMode:   "pass-fail",
			expected:   "You are a helpful baker.\n\nIMPORTANT: Start your response with exactly one of these words: PASS or FAIL. Be direct and clear.",
		},
		{
			name:       "Sentiment mode with empty base prompt",
			basePrompt: "",
			exitMode:   "sentiment",
			expected:   "IMPORTANT: Start your response with exactly one of these words: POSITIVE, NEGATIVE, or NEUTRAL. Be direct and clear.",
		},
		{
			name:       "No exit mode",
			basePrompt: "You are helpful",
			exitMode:   "",
			expected:   "You are helpful",
		},
		{
			name:       "Unknown exit mode",
			basePrompt: "You are helpful",
			exitMode:   "unknown",
			expected:   "You are helpful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhanceSystemPromptForExitCode(tt.basePrompt, tt.exitMode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineCustomExitCode(t *testing.T) {
	// create mock app with logger and config
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	// define test exit code maps
	testConfig := &config.Config{
		ExitCodes: map[string]config.ExitCodeMap{
			"bakery": {
				Description: "Bakery workflow steps",
				Rules: []config.ExitCodeRule{
					{MatchType: "exact", Pattern: "MIX", ExitCode: 100},
					{MatchType: "exact", Pattern: "BAKE", ExitCode: 101},
					{MatchType: "prefix", Pattern: "COOL", ExitCode: 102},
				},
			},
			"priority_order": {
				Description: "Tests rule priority - first match wins",
				Rules: []config.ExitCodeRule{
					{MatchType: "contains", Pattern: "test", ExitCode: 300},
					{MatchType: "contains", Pattern: "test completed", ExitCode: 301}, // should not match if "test" matches first
				},
			},
			"invalid_regex": {
				Description: "Contains invalid regex pattern",
				Rules: []config.ExitCodeRule{
					{MatchType: "regex", Pattern: "[invalid", ExitCode: 400}, // invalid regex
					{MatchType: "exact", Pattern: "fallback", ExitCode: 401},
				},
			},
			// Default config examples - these match the actual default_config.toml
			"approval": {
				Description: "Exits based on a three-stage approval status.",
				Rules: []config.ExitCodeRule{
					{MatchType: "contains", Pattern: "APPROVED", ExitCode: 20},
					{MatchType: "contains", Pattern: "REJECTED", ExitCode: 21},
					{MatchType: "contains", Pattern: "NEEDS_REVIEW", ExitCode: 22},
				},
			},
			"priority": {
				Description: "Exit codes based on priority levels.",
				Rules: []config.ExitCodeRule{
					{MatchType: "regex", Pattern: "(?i)\\b(urgent|critical|high)\\b", ExitCode: 30},
					{MatchType: "regex", Pattern: "(?i)\\b(medium|normal)\\b", ExitCode: 31},
					{MatchType: "regex", Pattern: "(?i)\\b(low|minor)\\b", ExitCode: 32},
				},
			},
		},
	}

	app := &App{
		cfg:    testConfig,
		logger: logger,
	}

	tests := []struct {
		name            string
		response        string
		exitCodeMapName string
		expected        int
	}{
		// bakery workflow example tests
		{
			name:            "Bakery MIX step",
			response:        "MIX",
			exitCodeMapName: "bakery",
			expected:        100,
		},
		{
			name:            "Bakery BAKE step",
			response:        "BAKE",
			exitCodeMapName: "bakery",
			expected:        101,
		},
		{
			name:            "Bakery COOL step",
			response:        "COOL for 20 minutes", // prefix
			exitCodeMapName: "bakery",
			expected:        102,
		},
		{
			name:            "Bakery step in sentence",
			response:        "Next step is to BAKE the cake",
			exitCodeMapName: "bakery",
			expected:        0, // exact match required
		},
		{
			name:            "Bakery no match",
			response:        "SERVE",
			exitCodeMapName: "bakery",
			expected:        0,
		},
		// Priority order tests
		{
			name:            "First rule wins",
			response:        "test completed successfully",
			exitCodeMapName: "priority_order",
			expected:        300, // Should match first "test" rule, not "test completed"
		},

		// Error handling tests
		{
			name:            "Invalid regex pattern",
			response:        "This should trigger invalid regex",
			exitCodeMapName: "invalid_regex",
			expected:        0, // Should skip invalid regex and not match anything
		},
		{
			name:            "Invalid regex with fallback match",
			response:        "fallback",
			exitCodeMapName: "invalid_regex",
			expected:        401, // Should skip invalid regex but match exact pattern
		},
		{
			name:            "Empty map name",
			response:        "MIX",
			exitCodeMapName: "",
			expected:        0,
		},
		{
			name:            "Non-existent map",
			response:        "MIX",
			exitCodeMapName: "non_existent",
			expected:        0,
		},

		// Default config examples verification
		{
			name:            "Approval APPROVED",
			response:        "The deployment has been APPROVED for production",
			exitCodeMapName: "approval",
			expected:        20,
		},
		{
			name:            "Approval REJECTED",
			response:        "Request REJECTED due to security concerns",
			exitCodeMapName: "approval",
			expected:        21,
		},
		{
			name:            "Approval NEEDS_REVIEW",
			response:        "Code NEEDS_REVIEW before deployment",
			exitCodeMapName: "approval",
			expected:        22,
		},
		{
			name:            "Approval no match",
			response:        "Status is pending further analysis",
			exitCodeMapName: "approval",
			expected:        0,
		},

		// Priority level tests
		{
			name:            "Priority urgent",
			response:        "This is an urgent security vulnerability",
			exitCodeMapName: "priority",
			expected:        30,
		},
		{
			name:            "Priority critical case insensitive",
			response:        "CRITICAL system failure detected",
			exitCodeMapName: "priority",
			expected:        30,
		},
		{
			name:            "Priority high",
			response:        "High priority bug needs immediate attention",
			exitCodeMapName: "priority",
			expected:        30,
		},
		{
			name:            "Priority medium",
			response:        "This has medium priority for next sprint",
			exitCodeMapName: "priority",
			expected:        31,
		},
		{
			name:            "Priority normal",
			response:        "Normal maintenance task",
			exitCodeMapName: "priority",
			expected:        31,
		},
		{
			name:            "Priority low",
			response:        "Low priority enhancement request",
			exitCodeMapName: "priority",
			expected:        32,
		},
		{
			name:            "Priority minor",
			response:        "Minor documentation update needed",
			exitCodeMapName: "priority",
			expected:        32,
		},
		{
			name:            "Priority no match",
			response:        "Status unknown at this time",
			exitCodeMapName: "priority",
			expected:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.determineCustomExitCode(tt.response, tt.exitCodeMapName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
