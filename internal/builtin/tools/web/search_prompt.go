package web

import "github.com/vesvai/vesvai/internal/agent/prompt"

func searchToolPromptBuilder() *prompt.Prompt {
	return prompt.New().
		List("Allows vesvai to search the web and use the results to inform responses",
			"Provides up-to-date information for current events and recent data",
			"Returns search result information formatted as search result blocks",
			"Use this tool for accessing information beyond your knowledge cutoff",
			"Searches are performed automatically within a single API call").
		Paragraph("Usage:").
		List("Domain filtering is supported to include or block specific websites.",
			"Web search is only available in the US")
}
