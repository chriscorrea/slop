package format

import (
	"testing"

	"slop/internal/config"
)

func TestCleanJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean JSON object",
			input:    `{"four_legs": "good", "year": 1945}`,
			expected: `{"four_legs": "good", "year": 1945}`,
		},
		{
			name:     "JSON with prefix text",
			input:    `Here's the JSON response: {"four_legs": "good"}`,
			expected: `{"four_legs": "good"}`,
		},
		{
			name:     "JSON with suffix text",
			input:    `{"two_legs": "better"} That's the response.`,
			expected: `{"two_legs": "better"}`,
		},
		{
			name:     "JSON with both prefix and suffix",
			input:    `Response: {"four_legs": "good"} End of output.`,
			expected: `{"four_legs": "good"}`,
		},
		{
			name:     "JSON array",
			input:    `[{"id": 1}, {"id": 2}]`,
			expected: `[{"id": 1}, {"id": 2}]`,
		},
		{
			name:     "JSON in markdown fence with newline",
			input:    "Here's the JSON:\n```json\n{\"two_legs\": \"good\"}\n```\nThat's it.",
			expected: `{"two_legs": "good"}`,
		},
		{
			name:     "JSON in markdown fence without newline",
			input:    "```json{\"two_legs\": \"good\"}```",
			expected: `{"two_legs": "good"}`,
		},
		{
			name:     "JSON in markdown fence with extra content",
			input:    "Some explanation:\n```json\n{\n  \"name\": \"test\",\n  \"value\": 123\n}\n```\nMore text after.",
			expected: "{\n  \"name\": \"test\",\n  \"value\": 123\n}",
		},
		{
			name:     "Multiple JSON blocks - fence takes priority",
			input:    "First: {\"a\": 1} then ```json\n{\"b\": 2}\n``` and {\"c\": 3}",
			expected: `{"b": 2}`,
		},
		{
			name:     "No JSON found",
			input:    `This is just plain text without any JSON.`,
			expected: `This is just plain text without any JSON.`,
		},
		{
			name:     "Nested JSON objects",
			input:    `{"outer": {"inner": {"deep": "value"}}}`,
			expected: `{"outer": {"inner": {"deep": "value"}}}`,
		},
		{
			name:     "Malformed JSON fallback",
			input:    `Some text {"incomplete": } more text`,
			expected: `{"incomplete": }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanJSON(tt.input)
			if result != tt.expected {
				t.Errorf("CleanJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean YAML",
			input:    "key: value\nnumber: 42",
			expected: "key: value\nnumber: 42",
		},
		{
			name:     "YAML with prefix text",
			input:    "Here's the YAML:\nkey: value\nnumber: 42",
			expected: "key: value\nnumber: 42",
		},
		{
			name:     "YAML list",
			input:    "- item1\n- item2\n- item3",
			expected: "- item1\n- item2\n- item3",
		},
		{
			name:     "YAML in markdown fence with newline",
			input:    "Configuration:\n```yaml\nname: test\nversion: 1.0\n```\nEnd.",
			expected: "name: test\nversion: 1.0",
		},
		{
			name:     "YAML in markdown fence without newline",
			input:    "```yamlname: test\nversion: 1.0```",
			expected: "name: test\nversion: 1.0",
		},
		{
			name:     "Complex YAML with nested structure",
			input:    "```yaml\nserver:\n  host: localhost\n  port: 8080\ndatabase:\n  driver: postgres\n  url: localhost:5432\n```",
			expected: "server:\n  host: localhost\n  port: 8080\ndatabase:\n  driver: postgres\n  url: localhost:5432",
		},
		{
			name:     "YAML with markdown fence takes priority",
			input:    "key1: value1\n```yaml\nkey2: value2\n```\nkey3: value3",
			expected: "key2: value2",
		},
		{
			name:     "No YAML found",
			input:    "This is just plain text.",
			expected: "This is just plain text.",
		},
		{
			name:     "YAML list in fence",
			input:    "Items:\n```yaml\n- first\n- second\n- third\n```",
			expected: "- first\n- second\n- third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanYAML(tt.input)
			if result != tt.expected {
				t.Errorf("CleanYAML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain markdown",
			input:    "# Title\n\nSome content.",
			expected: "# Title\n\nSome content.",
		},
		{
			name:     "Markdown in fence",
			input:    "Here's the markdown:\n```markdown\n# Header\n\nContent here.\n```\nDone.",
			expected: "# Header\n\nContent here.",
		},
		{
			name:     "Markdown in md fence",
			input:    "```md\n## Subtitle\n\n- List item\n```",
			expected: "## Subtitle\n\n- List item",
		},
		{
			name:     "Markdown in generic fence",
			input:    "```\n# Title\n\nParagraph text.\n```",
			expected: "# Title\n\nParagraph text.",
		},
		{
			name:     "Complex markdown with code blocks",
			input:    "```markdown\n# API Documentation\n\n## Usage\n\n```python\nprint('hello')\n```\n\nMore text.\n```",
			expected: "# API Documentation\n\n## Usage\n\n```python\nprint('hello')\n```\n\nMore text.",
		},
		{
			name:     "Nested fences - uses last closing fence",
			input:    "```markdown\n# Title\n\n```json\n{\"test\": true}\n```\n\nMore markdown.\n```",
			expected: "# Title\n\n```json\n{\"test\": true}\n```\n\nMore markdown.",
		},
		{
			name:     "Multiple markdown fences - first one wins",
			input:    "```markdown\nFirst block\n```\n\n```md\nSecond block\n```",
			expected: "First block\n```\n\n```md\nSecond block",
		},
		{
			name:     "No fence found",
			input:    "Regular markdown content without fences.\n\n# Header\n\nContent.",
			expected: "Regular markdown content without fences.\n\n# Header\n\nContent.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean XML",
			input:    "<root><item>value</item></root>",
			expected: "<root><item>value</item></root>",
		},
		{
			name:     "XML with prefix text",
			input:    "Here's the XML: <root><item>truffle</item></root>",
			expected: "<root><item>truffle</item></root>",
		},
		{
			name:     "XML in markdown fence",
			input:    "```xml\n<config>\n  <setting>value</setting>\n</config>\n```",
			expected: "<config>\n  <setting>value</setting>\n</config>",
		},
		{
			name:     "XML with comments",
			input:    "```xml\n<!-- Configuration -->\n<root>\n  <item>test</item>\n</root>\n```",
			expected: "<!-- Configuration -->\n<root>\n  <item>test</item>\n</root>",
		},
		{
			name:     "XML with processing instruction",
			input:    "```xml\n<?xml version=\"1.0\"?>\n<root/>\n```",
			expected: "<?xml version=\"1.0\"?>\n<root/>",
		},
		{
			name:     "Self-closing tags",
			input:    "<config><setting name=\"test\" value=\"123\"/></config>",
			expected: "<config><setting name=\"test\" value=\"123\"/></config>",
		},
		{
			name:     "Multiple XML elements - fence takes priority",
			input:    "<first/> then ```xml\n<second/>\n``` and <third/>",
			expected: "<second/>",
		},
		{
			name:     "XML fallback without fence",
			input:    "Some text < 5 and <item>content</item> more text",
			expected: "<item>content</item>",
		},
		{
			name:     "No XML found",
			input:    "Just plain text with no XML tags.",
			expected: "Just plain text with no XML tags.",
		},
		{
			name:     "Comparison operators ignored",
			input:    "if (x < 5 && y > 3) then <result>success</result>",
			expected: "<result>success</result>",
		},
		{
			name:     "Complex XML document",
			input:    "```xml\n<document>\n  <header id=\"1\">\n    <title>Test</title>\n  </header>\n  <body>\n    <p>Content</p>\n  </body>\n</document>\n```",
			expected: "<document>\n  <header id=\"1\">\n    <title>Test</title>\n  </header>\n  <body>\n    <p>Content</p>\n  </body>\n</document>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanXML(tt.input)
			if result != tt.expected {
				t.Errorf("CleanXML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		format   config.Format
		expected string
	}{
		{
			name:     "JSON format",
			input:    "Here's your JSON: ```json\n{\"four_legs\": \"better\"}\n```",
			format:   config.Format{JSON: true},
			expected: `{"four_legs": "better"}`,
		},
		{
			name:     "YAML format",
			input:    "Configuration:\n```yaml\nfour_legs: better\n```",
			format:   config.Format{YAML: true},
			expected: "four_legs: better",
		},
		{
			name:     "Markdown format",
			input:    "```markdown\n# Seven Commandments\n\nContent\n```",
			format:   config.Format{MD: true},
			expected: "# Seven Commandments\n\nContent",
		},
		{
			name:     "XML format",
			input:    "```xml\n<root><item/></root>\n```",
			format:   config.Format{XML: true},
			expected: "<root><item/></root>",
		},
		{
			name:     "No format specified",
			input:    "Just regular text with no special formatting.",
			format:   config.Format{},
			expected: "Just regular text with no special formatting.",
		},
		{
			name:     "Multiple formats false",
			input:    "Regular content that shouldn't be processed.",
			format:   config.Format{JSON: false, YAML: false, MD: false, XML: false},
			expected: "Regular content that shouldn't be processed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanResponse(tt.input, tt.format)
			if result != tt.expected {
				t.Errorf("CleanResponse() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMarkdownFenceEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) string
		input    string
		expected string
	}{
		{
			name:     "JSON - incomplete block fence",
			function: CleanJSON,
			input:    "```json\n{\"key\": \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "JSON - fence does not closing",
			function: CleanJSON,
			input:    "```json\n{\"incomplete\": true",
			expected: "```json\n{\"incomplete\": true",
		},
		{
			name:     "YAML - fence with extra backticks",
			function: CleanYAML,
			input:    "````yaml\nkey: value\n````",
			expected: "key: value",
		},
		{
			name:     "Markdown - empty fence",
			function: CleanMarkdown,
			input:    "```markdown\n\n```",
			expected: "",
		},
		{
			name:     "JSON - multiple fences nested",
			function: CleanJSON,
			input:    "```json\n{\"outer\": \"```json inside string```\"}\n```",
			expected: "{\"outer\": \"",
		},
		{
			name:     "YAML - fence with other content",
			function: CleanYAML,
			input:    "Here's some text:\n```yaml\ntest: value\n```\nAnd here's more:\n```json\n{\"not\": \"yaml\"}\n```",
			expected: "test: value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			if result != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}
