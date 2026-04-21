package collector

import (
	"encoding/json"
	"strings"
)

func sanitizePromptDomain(domain, raw string) string {
	switch domain {
	case "calendar":
		return sanitizeCalendarPromptJSON(raw)
	case "vc":
		return sanitizeVCPromptJSON(raw)
	case "mail":
		return sanitizeMailPromptJSON(raw)
	case "docs_content":
		return sanitizeDocsContentPromptJSON(raw)
	case "mail_content", "vc_notes":
		return sanitizePromptWrapperJSON(raw)
	default:
		return raw
	}
}

func sanitizeCalendarPromptJSON(raw string) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return raw
	}
	delete(root, "_notice")
	container, ok := root["data"].([]any)
	if !ok {
		return marshalPromptJSON(root, raw)
	}
	filtered := make([]any, 0, len(container))
	for _, item := range container {
		event, ok := item.(map[string]any)
		if !ok {
			continue
		}
		trimmed := map[string]any{}
		copyIfString(trimmed, "summary", event)
		copyIfString(trimmed, "description", event)
		copyIfMap(trimmed, "start_time", event)
		copyIfMap(trimmed, "end_time", event)
		copyIfString(trimmed, "self_rsvp_status", event)
		copyIfString(trimmed, "free_busy_status", event)
		if organizer, ok := event["event_organizer"].(map[string]any); ok {
			trimmedOrganizer := map[string]any{}
			copyIfString(trimmedOrganizer, "display_name", organizer)
			if len(trimmedOrganizer) > 0 {
				trimmed["event_organizer"] = trimmedOrganizer
			}
		}
		if len(trimmed) > 0 {
			filtered = append(filtered, trimmed)
		}
	}
	root["data"] = filtered
	return marshalPromptJSON(root, raw)
}

func sanitizeVCPromptJSON(raw string) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return raw
	}
	delete(root, "_notice")
	data, ok := root["data"].(map[string]any)
	if !ok {
		return marshalPromptJSON(root, raw)
	}
	trimmedData := map[string]any{}
	if hasMore, ok := data["has_more"].(bool); ok {
		trimmedData["has_more"] = hasMore
	}
	items, _ := data["items"].([]any)
	filteredItems := make([]any, 0, len(items))
	for _, item := range items {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		trimmed := map[string]any{}
		copyIfString(trimmed, "display_info", asMap)
		if meta, ok := asMap["meta_data"].(map[string]any); ok {
			trimmedMeta := map[string]any{}
			copyIfString(trimmedMeta, "description", meta)
			if len(trimmedMeta) > 0 {
				trimmed["meta_data"] = trimmedMeta
			}
		}
		if len(trimmed) > 0 {
			filteredItems = append(filteredItems, trimmed)
		}
	}
	trimmedData["items"] = filteredItems
	root["data"] = trimmedData
	return marshalPromptJSON(root, raw)
}

func sanitizeMailPromptJSON(raw string) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	items, ok := payload.([]any)
	if !ok {
		return raw
	}
	filtered := make([]any, 0, len(items))
	for _, item := range items {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		trimmed := map[string]any{}
		copyIfString(trimmed, "date", asMap)
		copyIfString(trimmed, "from", asMap)
		copyIfString(trimmed, "subject", asMap)
		copyIfString(trimmed, "labels", asMap)
		if len(trimmed) > 0 {
			filtered = append(filtered, trimmed)
		}
	}
	return marshalPromptJSON(filtered, raw)
}

func sanitizeDocsContentPromptJSON(raw string) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return raw
	}
	delete(root, "_notice")
	data, ok := root["data"].(map[string]any)
	if !ok {
		return marshalPromptJSON(root, raw)
	}
	trimmedData := map[string]any{}
	copyIfString(trimmedData, "title", data)
	copyIfString(trimmedData, "markdown", data)
	copyIfString(trimmedData, "message", data)
	root["data"] = trimmedData
	return marshalPromptJSON(root, raw)
}

func sanitizePromptWrapperJSON(raw string) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return raw
	}
	delete(root, "_notice")
	return marshalPromptJSON(root, raw)
}

func copyIfString(dst map[string]any, key string, src map[string]any) {
	if value := strings.TrimSpace(stringValue(src, key)); value != "" {
		dst[key] = value
	}
}

func copyIfMap(dst map[string]any, key string, src map[string]any) {
	value, ok := src[key].(map[string]any)
	if !ok || len(value) == 0 {
		return
	}
	dst[key] = value
}

func marshalPromptJSON(payload any, fallback string) string {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fallback
	}
	return string(data)
}
