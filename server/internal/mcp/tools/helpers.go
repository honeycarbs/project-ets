package tools

func stubResponse(tool, message string) map[string]any {
	return map[string]any{
		"status":  "not_implemented",
		"tool":    tool,
		"message": message,
	}
}
