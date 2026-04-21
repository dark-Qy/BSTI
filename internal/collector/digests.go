package collector

import (
	"encoding/json"
	"strings"
)

func BuildAnalysisInput(bundle Bundle, chatStats, mailStats DomainDigestStats) AnalysisInput {
	domains := groupDomainResults(bundle.Domains)
	stats := map[string]DomainDigestStats{
		"chat_prompt": chatStats,
		"mail_filter": withMailFetchedCount(mailStats, len(domains["mail_content"])),
	}
	for _, domain := range []string{"chat", "docs", "docs_content", "task", "calendar", "mail", "mail_content", "vc", "vc_notes"} {
		results := domains[domain]
		if len(results) == 0 {
			continue
		}
		stats[domain+"_raw"] = rawDomainStats(results)
	}
	if chatRaw, ok := stats["chat_raw"]; ok {
		chatRaw.RawChars = chatStats.RawChars
		stats["chat_raw"] = chatRaw
	}
	return AnalysisInput{Stats: stats}
}

func groupDomainResults(results []DomainResult) map[string][]DomainResult {
	grouped := make(map[string][]DomainResult)
	for _, result := range results {
		grouped[result.Domain] = append(grouped[result.Domain], result)
	}
	return grouped
}

func rawDomainStats(results []DomainResult) DomainDigestStats {
	stats := DomainDigestStats{FetchedItems: len(results)}
	for _, result := range results {
		stats.RawChars += len(result.Stdout)
	}
	return stats
}

func withMailFetchedCount(stats DomainDigestStats, fetched int) DomainDigestStats {
	stats.FetchedItems = fetched
	return stats
}

func filterMailDomain(result DomainResult) (DomainResult, DomainDigestStats) {
	stats := DomainDigestStats{RawChars: len(result.Stdout)}
	result.Stdout = filterMailJSON(result.Stdout, &stats)
	return result, stats
}

func filterMailJSON(raw string, stats *DomainDigestStats) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return raw
	}
	container := root
	key := "items"
	if dataMap, ok := root["data"].(map[string]any); ok {
		container = dataMap
	}
	items, ok := container[key].([]any)
	if !ok {
		return raw
	}

	filtered := make([]any, 0, len(items))
	for _, item := range items {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		stats.ItemsBefore++
		combined := strings.Join([]string{
			stringValue(asMap, "subject", "title", "name", "summary", "snippet", "preview"),
			stringValue(asMap, "from", "from_email", "sender", "sender_email", "email", "address"),
		}, "\n")
		if looksLikeSystemMail(combined) {
			continue
		}
		filtered = append(filtered, asMap)
		stats.ItemsAfter++
	}
	container[key] = filtered
	stats.FilteredItems = max(0, stats.ItemsBefore-stats.ItemsAfter)
	data, err := json.Marshal(root)
	if err != nil {
		return raw
	}
	return string(data)
}

func filterMailByBodies(mailResult DomainResult, contents []DomainResult) (DomainResult, []DomainResult) {
	blockedIDs := make(map[string]struct{})
	filteredContents := make([]DomainResult, 0, len(contents))
	for _, result := range contents {
		if looksLikeSystemMail(result.Stdout) {
			if messageID := messageIDFromMailContentCommand(result.Command); messageID != "" {
				blockedIDs[messageID] = struct{}{}
			}
			continue
		}
		filteredContents = append(filteredContents, result)
	}
	if len(blockedIDs) == 0 {
		return mailResult, filteredContents
	}
	mailResult.Stdout = removeMailItemsByID(mailResult.Stdout, blockedIDs)
	return mailResult, filteredContents
}

func messageIDFromMailContentCommand(command string) string {
	parts := strings.Fields(command)
	for index := 0; index < len(parts)-1; index++ {
		if parts[index] == "--message-id" {
			return strings.TrimSpace(parts[index+1])
		}
	}
	return ""
}

func removeMailItemsByID(raw string, blockedIDs map[string]struct{}) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return raw
	}
	container := root
	key := "items"
	if dataMap, ok := root["data"].(map[string]any); ok {
		container = dataMap
	}
	items, ok := container[key].([]any)
	if !ok {
		return raw
	}
	filtered := make([]any, 0, len(items))
	for _, item := range items {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if messageID := stringValue(asMap, "message_id", "id"); messageID != "" {
			if _, blocked := blockedIDs[messageID]; blocked {
				continue
			}
		}
		filtered = append(filtered, asMap)
	}
	container[key] = filtered
	data, err := json.Marshal(root)
	if err != nil {
		return raw
	}
	return string(data)
}

func looksLikeSystemMail(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"noreply", "no-reply", "notification", "github", "验证码", "verification code", "bot", "系统通知", "自动发送",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
