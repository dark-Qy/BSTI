package collector

import (
	"context"
	"encoding/json"
	"strings"
)

type docsSearchOutcome struct {
	args   []string
	raw    string
	stderr string
	err    error
}

func (c *Collector) collectDocsDomain(ctx context.Context, sessionDir, startISO, endISO string) (DomainResult, []DomainResult, DocsSummary, error) {
	result, summary, refs, err := c.searchDocsDomain(ctx, sessionDir, startISO, endISO)
	if err != nil {
		return DomainResult{}, nil, DocsSummary{}, err
	}
	extras, err := c.runExtraFetches(ctx, "docs_content", refs, func(ref string) []string {
		return []string{"docs", "+fetch", "--doc", ref, "--format", "json"}
	})
	if err != nil {
		return result, nil, DocsSummary{}, err
	}
	return result, extras, summary, nil
}

func (c *Collector) searchDocsDomain(ctx context.Context, sessionDir, startISO, endISO string) (DomainResult, DocsSummary, []string, error) {
	userOpenID, err := loadSessionUserOpenID(sessionDir)
	if err != nil {
		return DomainResult{
			Domain:  "docs",
			Command: "resolve session user open id",
			Error:   err.Error(),
		}, DocsSummary{}, nil, nil
	}

	personalFilter, _ := json.Marshal(map[string]any{
		"creator_ids": []string{userOpenID},
		"sort_rule":   "EDIT_TIME",
	})
	recentFilter, _ := json.Marshal(map[string]any{
		"open_time": map[string]string{"start": startISO, "end": endISO},
		"sort_rule": "OPEN_TIME",
	})

	personalArgs := []string{"docs", "+search", "--filter", string(personalFilter), "--page-size", "10", "--format", "json"}
	recentArgs := []string{"docs", "+search", "--filter", string(recentFilter), "--page-size", "20", "--format", "json"}
	searches := runBoundedOrdered(ctx, []docsSearchOutcome{
		{args: personalArgs},
		{args: recentArgs},
	}, extraFetchConcurrency, func(ctx context.Context, input docsSearchOutcome) docsSearchOutcome {
		out, err := c.runner.Run(ctx, input.args)
		return docsSearchOutcome{
			args:   input.args,
			raw:    out.Stdout,
			stderr: out.Stderr,
			err:    err,
		}
	})
	personalSearch := searches[0]
	recentSearch := searches[1]

	aggregated := map[string]any{
		"personal_search": json.RawMessage(personalSearch.raw),
		"recent_search":   json.RawMessage(recentSearch.raw),
	}
	stdout, _ := json.Marshal(aggregated)
	result := DomainResult{
		Domain:  "docs",
		Command: strings.Join(personalArgs, " ") + " || " + strings.Join(recentArgs, " "),
		Stdout:  string(stdout),
		Stderr:  strings.TrimSpace(strings.Join([]string{personalSearch.stderr, recentSearch.stderr}, "\n")),
		Error:   joinErrors(errorString(personalSearch.err), errorString(recentSearch.err)),
	}

	personalDocs := extractDocSummaries(personalSearch.raw, "created_by_self")
	recentDocs := extractDocSummaries(recentSearch.raw, "recent_open")
	selected := mergeDocSummaries(personalDocs, recentDocs)
	refs := make([]string, 0, len(personalDocs))
	for _, doc := range takeFirstDocs(personalDocs, 10) {
		refs = append(refs, firstNonEmpty(doc.URL, doc.Identifier))
	}
	return result, DocsSummary{
		SelectionRule:     "先取当前用户创建的文档 10 篇，再补最近浏览 20 篇，合并去重。",
		ApproximationNote: "当前通过 creator_ids=[当前用户 open_id] 获取当前用户创建的文档。",
		PersonalDocs:      personalDocs,
		RecentDocs:        recentDocs,
		SelectedDocs:      selected,
	}, refs, nil
}

func extractDocSummaries(raw string, reason string) []DocSummary {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	resultItems := extractDocResultItems(payload)
	docs := make([]DocSummary, 0, len(resultItems))
	for _, item := range resultItems {
		url := firstNonEmpty(
			nestedStringValue(item, "result_meta", "url"),
			stringValue(item, "url"),
		)
		identifier := firstNonEmpty(
			identifierFromDocRef(url),
			nestedStringValue(item, "result_meta", "token"),
			stringValue(item, "obj_token", "doc_token", "token"),
		)
		if identifier == "" && url == "" {
			continue
		}
		title := firstNonEmpty(
			stringValue(item, "title_highlighted", "title", "name"),
			nestedStringValue(item, "result_meta", "title"),
		)
		docs = append(docs, DocSummary{
			Title:      firstNonEmpty(title, identifier),
			Identifier: identifier,
			URL:        url,
			Reason:     reason,
		})
	}
	return dedupeDocSummaries(docs)
}

func extractDocResultItems(payload any) []map[string]any {
	switch typed := payload.(type) {
	case []any:
		return mapsFromItems(typed)
	case map[string]any:
		if results, ok := typed["results"]; ok {
			if array, ok := results.([]any); ok {
				return mapsFromItems(array)
			}
		}
		if items, ok := typed["items"]; ok {
			if array, ok := items.([]any); ok {
				return mapsFromItems(array)
			}
		}
		if data, ok := typed["data"]; ok {
			switch nested := data.(type) {
			case []any:
				return mapsFromItems(nested)
			case map[string]any:
				if results, ok := nested["results"]; ok {
					if array, ok := results.([]any); ok {
						return mapsFromItems(array)
					}
				}
				if items, ok := nested["items"]; ok {
					if array, ok := items.([]any); ok {
						return mapsFromItems(array)
					}
				}
			}
		}
		return []map[string]any{typed}
	default:
		return nil
	}
}

func mapsFromItems(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if typed, ok := item.(map[string]any); ok {
			out = append(out, typed)
		}
	}
	return out
}

func nestedStringValue(item map[string]any, path ...string) string {
	if len(path) == 0 {
		return ""
	}

	current := any(item)
	for _, key := range path[:len(path)-1] {
		asMap, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		next, ok := asMap[key]
		if !ok {
			return ""
		}
		current = next
	}

	asMap, ok := current.(map[string]any)
	if !ok {
		return ""
	}
	raw, ok := asMap[path[len(path)-1]]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func mergeDocSummaries(groups ...[]DocSummary) []DocSummary {
	merged := make([]DocSummary, 0)
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, doc := range group {
			key := firstNonEmpty(doc.Identifier, doc.URL, doc.Title)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, doc)
		}
	}
	return merged
}

func dedupeDocSummaries(items []DocSummary) []DocSummary {
	return mergeDocSummaries(items)
}

func identifierFromDocRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	ref = strings.TrimRight(ref, "/")
	if idx := strings.LastIndex(ref, "/"); idx >= 0 && idx < len(ref)-1 {
		return ref[idx+1:]
	}
	return ref
}

func takeFirstDocs(values []DocSummary, limit int) []DocSummary {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func joinErrors(values ...string) string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	return strings.Join(filtered, "\n")
}
