package format

import (
	"encoding/json"
	"regexp"
	"strings"

	"slop/internal/config"
)

// CleanResponse removes text outside format boundaries
func CleanResponse(response string, format config.Format) string {
	if format.JSON {
		return CleanJSON(response)
	}

	if format.JSONL {
		return CleanJSONL(response)
	}

	if format.YAML {
		return CleanYAML(response)
	}

	if format.MD {
		return CleanMarkdown(response)
	}

	if format.XML {
		return CleanXML(response)
	}

	return response
}

// CleanJSON extracts JSON, prioritizing markdown backtick fences
func CleanJSON(response string) string {
	// check for a JSON markdown code block
	if strings.Contains(response, "```json") {
		// isolate the content within json block
		startMarker := "```json\n"
		endMarker := "```"

		startIdx := strings.Index(response, startMarker)
		if startIdx == -1 {
			// if ```json is present but the newline is missing, adapt the search
			startMarker = "```json"
			startIdx = strings.Index(response, startMarker)
		}

		// find the content after the start marker
		contentAfterStart := response[startIdx+len(startMarker):]
		endIdx := strings.Index(contentAfterStart, endMarker)

		if endIdx != -1 {
			// return the clean JSON content found inside the block
			return strings.TrimSpace(contentAfterStart[:endIdx])
		}
	}

	// fallback for other responses: first '{' or '[' and the last '}' or ']'
	startIdx := strings.IndexAny(response, "[{")
	if startIdx == -1 {
		return response
	}

	endIdx := strings.LastIndexAny(response, "}]")
	if endIdx == -1 || endIdx < startIdx {
		return response
	}

	return response[startIdx : endIdx+1]
}

// CleanYAML extracts YAML, prioritizing markdown code fence
func CleanYAML(response string) string {
	// first, check for a YAML markdown backtick block
	yamlFenceRegex := regexp.MustCompile("```+yaml\n?")
	if yamlFenceRegex.MatchString(response) {
		match := yamlFenceRegex.FindStringIndex(response)
		startIdx := match[1] // pos after the opening ```

		// find the content after the start marker
		contentAfterStart := response[startIdx:]

		// look for closing ```
		endFenceRegex := regexp.MustCompile("```+")
		endMatch := endFenceRegex.FindStringIndex(contentAfterStart)

		if endMatch != nil {
			endIdx := endMatch[0]
			// return the clean YAML content found inside the block
			return strings.TrimSpace(contentAfterStart[:endIdx])
		}
	}

	// fallback: find the first line that looks like key-value pair or list item
	lines := strings.Split(response, "\n")
	startIdx := -1

	// simple regex to find a line starting with a key: or a list item "- "
	yamlPattern := regexp.MustCompile(`^\s*[-\w]+\s*:|^\s*-\s`)
	for i, line := range lines {
		if yamlPattern.MatchString(line) {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return response // no clear YAML start found, return as-is
	}

	// if yaml start index is found, assume everything to end is YAML
	return strings.Join(lines[startIdx:], "\n")
}

// CleanMarkdown extracts Markdown from code fence if present
func CleanMarkdown(response string) string {
	markers := []string{"```markdown\n", "```md\n", "```\n"}
	endMarker := "```"

	for _, startMarker := range markers {
		if !strings.Contains(response, startMarker) {
			continue
		}

		startIdx := strings.Index(response, startMarker)

		// find the content that appears after the start marker
		contentAfterStart := response[startIdx+len(startMarker):]

		// use LastIndex to find the *final* closing fence, correctly handling nested blocks
		endIdx := strings.LastIndex(contentAfterStart, endMarker)

		if endIdx != -1 {
			// if block is found, return the content inside it
			return strings.TrimSpace(contentAfterStart[:endIdx])
		}
	}

	// if no block is found, return the original string as-is
	return response
}

// CleanXML extracts valid XML from response
func CleanXML(response string) string {
	// first, check for an XML markdown code block
	startMarker := "```xml\n"
	endMarker := "```"

	if strings.Contains(response, startMarker) {
		startIdx := strings.Index(response, startMarker)
		contentAfterStart := response[startIdx+len(startMarker):]

		// use LastIndex to correctly handle nested blocks or other fences
		endIdx := strings.LastIndex(contentAfterStart, endMarker)

		if endIdx != -1 {
			return strings.TrimSpace(contentAfterStart[:endIdx])
		}
	}

	// fallback is regex for plausible xml tags
	startTagRegex := regexp.MustCompile(`<([a-zA-Z!?/])`)

	match := startTagRegex.FindStringIndex(response)
	if match == nil {
		return response // No plausible XML tag found
	}

	// get the starting position of the first match found
	startIdx := match[0]

	// find the last closing '>' to complete the block
	endIdx := strings.LastIndex(response, ">")

	// basic validation to ensure closing tag after opening
	if endIdx == -1 || endIdx < startIdx {
		return response
	}

	return strings.TrimSpace(response[startIdx : endIdx+1])
}

// CleanJSONL extracts JSONL (JSON Lines), prioritizing markdown code fence
func CleanJSONL(response string) string {
	// first, check for a JSONL markdown code block
	if strings.Contains(response, "```jsonl") {
		startMarker := "```jsonl\n"
		endMarker := "```"

		startIdx := strings.Index(response, startMarker)
		if startIdx == -1 {
			// if ```jsonl is present but the newline is missing, adapt the search
			startMarker = "```jsonl"
			startIdx = strings.Index(response, startMarker)
		}

		// find the content after the start marker
		contentAfterStart := response[startIdx+len(startMarker):]
		endIdx := strings.Index(contentAfterStart, endMarker)

		if endIdx != -1 {
			// extract content from fence and validate each line
			fenceContent := strings.TrimSpace(contentAfterStart[:endIdx])
			return validateAndFilterJSONL(fenceContent)
		}
	}

	// fallback: process the entire response line by line
	return validateAndFilterJSONL(response)
}

// validateAndFilterJSONL processes text line by line, keeping only valid JSON lines
func validateAndFilterJSONL(text string) string {
	lines := strings.Split(text, "\n")
	var validLines []string

	for _, line := range lines {
		// trim whitespace from each line
		trimmedLine := strings.TrimSpace(line)

		// skip empty lines
		if trimmedLine == "" {
			continue
		}

		// validate that the line is valid JSON
		if json.Valid([]byte(trimmedLine)) {
			validLines = append(validLines, trimmedLine)
		}
	}

	// return the valid lines joined with newlines
	return strings.Join(validLines, "\n")
}
