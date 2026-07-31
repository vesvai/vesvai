package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/llm"
)

func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var providerErr *llm.ProviderError
	if errors.As(err, &providerErr) {
		return formatProviderError(providerErr)
	}

	errStr := err.Error()

	if strings.Contains(errStr, "request already in progress") {
		return "Please wait for the current response to finish."
	}

	if strings.Contains(errStr, "connection reset") {
		return "Connection was reset. Please try again."
	}

	if strings.Contains(errStr, "connection refused") {
		return "Cannot connect to the server. Please check your internet connection."
	}

	if strings.Contains(errStr, "deadline exceeded") || strings.Contains(errStr, "timeout") {
		return "Request timed out. Please try again."
	}

	if strings.Contains(errStr, "broken pipe") {
		return "Connection was lost. Please try again."
	}

	if strings.Contains(errStr, "EOF") {
		return "Connection was closed unexpectedly. Please try again."
	}

	if strings.Contains(errStr, "failed after") && strings.Contains(errStr, "retries") {
		return "Service is temporarily unavailable. Please try again in a moment."
	}

	return truncateError(errStr)
}

func truncateError(s string) string {
	const max = 300
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func formatProviderError(err *llm.ProviderError) string {
	statusCode := err.StatusCode

	switch {
	case statusCode == 401:
		return "Invalid API key. Please check your configuration."
	case statusCode == 403:
		return "Access denied. Please check your API permissions."
	case statusCode == 404:
		return "Model not found. Please select a different model."
	case statusCode == 429:
		return "Rate limit exceeded. Please wait a moment and try again."
	case statusCode >= 500:
		return "Server error. Please try again in a moment."
	default:
		return fmt.Sprintf("API error (HTTP %d): %s", statusCode, err.Message)
	}
}
