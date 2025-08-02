// Provides exit code functionality
//
// The exit code system supports automatic sentiment analysis and pass/fail detection
//
// To extend with additional languages:
// 1. Add new terms to the appropriate regex patterns below
// 2. Include language-specific positive/negative/neutral terms
// 3. Add corresponding test cases in exit_codes_test.go
//
// Note: all regex patterns use case-insensitive matching, UTF-8 is supported.
package app

import (
	"regexp"
	"strings"
	"sync"
)

// exit codes for built-in modes
const (
	ExitPositive = 10
	ExitNegative = 11
	ExitNeutral  = 12
	ExitPass     = 30
	ExitFail     = 31
)

// pre-compiled regex patterns for sentiment and pass/fail detection
// supporting latin scripts: English, Spanish, French, German, Portuguese, Italian, Dutch, Swedish/Norwegian/Danish
var sentimentRegex = struct {
	Positive *regexp.Regexp
	Negative *regexp.Regexp
	Neutral  *regexp.Regexp
}{
	// English and other major Latin-script languages
	Positive: regexp.MustCompile(`(?i)\b(positive|good|yes|great|excellent|wonderful|fantastic|success|approve|agreed|affirmative|` +
		// Spanish
		`bueno|bien|excelente|fantástico|sí|positivo|maravilloso|estupendo|` +
		// French
		`bon|bien|excellent|fantastique|oui|positif|merveilleux|formidable|` +
		// German
		`gut|ausgezeichnet|fantastisch|ja|positiv|wunderbar|hervorragend|` +
		// Portuguese
		`bom|bem|excelente|fantástico|sim|positivo|maravilhoso|ótimo|` +
		// Italian
		`buono|bene|eccellente|fantastico|sì|positivo|meraviglioso|ottimo|` +
		// Dutch
		`goed|uitstekend|fantastisch|ja|positief|geweldig|prima|` +
		// Swedish/Norwegian/Danish
		`bra|utmärkt|fantastisk|ja|positiv|underbar|fremragende)\b`),

	Negative: regexp.MustCompile(`(?i)\b(negative|bad|no|terrible|awful|horrible|fail|error|disapprove|disappointing|denied|` +
		// Spanish
		`malo|mal|terrible|horrible|no|negativo|pésimo|desastroso|` +
		// French
		`mauvais|mal|terrible|horrible|non|négatif|affreux|désastreux|` +
		// German
		`schlecht|schrecklich|furchtbar|nein|negativ|entsetzlich|katastrophal|` +
		// Portuguese
		`mau|mal|terrível|horrível|não|negativo|péssimo|desastroso|` +
		// Italian
		`cattivo|male|terribile|orribile|no|negativo|pessimo|disastroso|` +
		// Dutch
		`slecht|verschrikkelijk|vreselijk|nee|negatief|afschuwelijk|` +
		// Swedish/Norwegian/Danish
		`dålig|fruktansvärd|hemsk|nej|negativ|förskräcklig)\b`),

	Neutral: regexp.MustCompile(`(?i)\b(neutral|neither|unclear|ambiguous|uncertain|maybe|mixed|moderate|average|okay|` +
		// Spanish
		`neutral|neutro|incierto|promedio|regular|quizás|tal vez|` +
		// French
		`neutre|incertain|moyen|peut-être|modéré|` +
		// German
		`neutral|ungewiss|durchschnittlich|vielleicht|mittelmäßig|` +
		// Portuguese
		`neutro|incerto|médio|talvez|moderado|` +
		// Italian
		`neutro|incerto|medio|forse|moderato|` +
		// Dutch
		`neutraal|onzeker|gemiddeld|misschien|` +
		// Swedish/Norwegian/Danish
		`neutral|osäker|genomsnittlig|kanske|måske)\b`),
}

var passFailRegex = struct {
	Pass *regexp.Regexp
	Fail *regexp.Regexp
}{
	// English and other major Latin-script languages
	Pass: regexp.MustCompile(`(?i)\b(pass|passed|success|approved|accepted|ok|okay|true|correct|valid|successful|` +
		// Spanish
		`aprobado|aceptado|correcto|válido|exitoso|verdadero|` +
		// French
		`réussi|approuvé|accepté|correct|valide|succès|vrai|` +
		// German
		`bestanden|genehmigt|akzeptiert|korrekt|gültig|erfolgreich|richtig|` +
		// Portuguese
		`aprovado|aceito|correto|válido|sucesso|verdadeiro|` +
		// Italian
		`approvato|accettato|corretto|valido|successo|vero|` +
		// Dutch
		`geslaagd|goedgekeurd|geaccepteerd|correct|geldig|succesvol|` +
		// Swedish/Norwegian/Danish
		`godkänd|godkjent|accepterad|korrekt|giltig|framgångsrik)\b`),

	Fail: regexp.MustCompile(`(?i)(^|\s|[^\p{L}])(fail|failed|error|rejected|denied|false|incorrect|invalid|unsuccessful|` +
		// Spanish
		`fallar|fallado|rechazado|denegado|falso|incorrecto|inválido|fracaso|` +
		// French
		`échoué|rejeté|refusé|faux|incorrect|invalide|échec|erreur|` +
		// German
		`fehlgeschlagen|abgelehnt|verweigert|falsch|ungültig|fehler|` +
		// Portuguese
		`falhou|rejeitado|negado|falso|incorreto|inválido|fracasso|erro|` +
		// Italian
		`fallito|respinto|negato|falso|errato|invalido|fallimento|errore|` +
		// Dutch
		`mislukt|afgewezen|geweigerd|vals|onjuist|ongeldig|fout|` +
		// Swedish/Norwegian/Danish
		`misslyckades|avvisad|nekad|falsk|felaktig|ogiltig|fel)($|\s|[^\p{L}])`),
}

// regex cache for custom exit code patterns to avoid recompilation
var (
	regexCache sync.Map // map[string]*regexp.Regexp
)

// getCompiledRegex returns a cached compiled regex or compiles and caches a new one
func getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if cached, ok := regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCache.Store(pattern, re)
	return re, nil
}

// getTextPart extracts partial text based on length and direction
func getTextPart(text string, length int, fromEnd bool) string {
	if len(text) <= length {
		return text
	}

	if fromEnd {
		return text[len(text)-length:]
	}
	return text[:length]
}

// determineSentimentExitCode performs a multi-pass check for sentiment
func determineSentimentExitCode(response string) int {
	// first pass: strict check on the beginning
	firstPart := getTextPart(response, 15, false) // first 15 characters
	if sentimentRegex.Positive.MatchString(firstPart) {
		return ExitPositive
	}
	if sentimentRegex.Negative.MatchString(firstPart) {
		return ExitNegative
	}
	if sentimentRegex.Neutral.MatchString(firstPart) {
		return ExitNeutral
	}

	// second pass: check the end
	lastPart := getTextPart(response, 15, true) // last 15 characters
	if sentimentRegex.Positive.MatchString(lastPart) {
		return ExitPositive
	}
	if sentimentRegex.Negative.MatchString(lastPart) {
		return ExitNegative
	}
	if sentimentRegex.Neutral.MatchString(lastPart) {
		return ExitNeutral
	}

	// third pass: broader check on the beginning
	introPart := getTextPart(response, 25, false) // first 25 characters
	if sentimentRegex.Positive.MatchString(introPart) {
		return ExitPositive
	}
	if sentimentRegex.Negative.MatchString(introPart) {
		return ExitNegative
	}
	if sentimentRegex.Neutral.MatchString(introPart) {
		return ExitNeutral
	}

	return 0 // no match
}

// determinePassFailExitCode performs a multi-pass check for pass/fail
func determinePassFailExitCode(response string) int {
	// first pass: strict check on the beginning
	firstPart := getTextPart(response, 15, false)
	if passFailRegex.Pass.MatchString(firstPart) {
		return ExitPass
	}
	if passFailRegex.Fail.MatchString(firstPart) {
		return ExitFail
	}

	// second pass: check the end
	lastPart := getTextPart(response, 15, true)
	if passFailRegex.Pass.MatchString(lastPart) {
		return ExitPass
	}
	if passFailRegex.Fail.MatchString(lastPart) {
		return ExitFail
	}

	// third pass: broader check on the beginning
	introPart := getTextPart(response, 25, false)
	if passFailRegex.Pass.MatchString(introPart) {
		return ExitPass
	}
	if passFailRegex.Fail.MatchString(introPart) {
		return ExitFail
	}

	return 0 // no match
}

// determineCustomExitCode handles user-defined maps from the config file
func (a *App) determineCustomExitCode(response string, exitCodeMapName string) int {
	if exitCodeMapName == "" {
		return 0
	}
	exitMap, ok := a.cfg.ExitCodes[exitCodeMapName]
	if !ok {
		a.logger.Warn("Specified exit code map not found", "map_name", exitCodeMapName)
		return 0
	}
	for _, rule := range exitMap.Rules {
		match := false
		switch rule.MatchType {
		case "exact":
			match = (response == rule.Pattern)
		case "contains":
			match = strings.Contains(response, rule.Pattern)
		case "prefix":
			match = strings.HasPrefix(response, rule.Pattern)
		case "suffix":
			match = strings.HasSuffix(response, rule.Pattern)
		case "regex":
			re, err := getCompiledRegex(rule.Pattern)
			if err != nil {
				a.logger.Error("Invalid regex in exit code rule", "pattern", rule.Pattern, "error", err)
				continue
			}
			match = re.MatchString(response)
		}
		if match {
			a.logger.Debug("Exit code rule matched", "pattern", rule.Pattern, "exit_code", rule.ExitCode)
			return rule.ExitCode
		}
	}
	return 0 // no match
}
