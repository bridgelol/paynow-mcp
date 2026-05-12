package paynow

func objectSchema(required []string, properties map[string]any) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func emptySchema() map[string]any {
	return objectSchema(nil, map[string]any{})
}

func stringProperty(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func boolProperty(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}

func enumProperty(description string, values ...string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
		"enum":        values,
	}
}

func queryProperty() map[string]any {
	return map[string]any{
		"type":                 "object",
		"description":          "Query string parameters to send to PayNow.",
		"additionalProperties": true,
	}
}

func bodyProperty(description string) map[string]any {
	return map[string]any{
		"description": description,
	}
}
