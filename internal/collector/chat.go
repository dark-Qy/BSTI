package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const groupMemberDisplayLimit = 10
const p2pTriggerBlockLimit = 2
const groupInitiationLookback = 3

var noisyGroupKeywords = []string{
	"闲聊", "唠嗑", "吃饭", "拼饭", "饭搭子", "奶茶", "下午茶", "外卖", "团建",
	"八卦", "游戏", "狼人杀", "旅游", "追剧", "电影", "唱k", "karaoke",
	"casual", "social", "meal", "lunch", "dinner", "gaming", "game", "entertainment",
}

type messageCandidate struct {
	MessageID     string
	ChatID        string
	ChatName      string
	ChatType      string
	CounterpartID string
	SenderID      string
	SenderName    string
	Text          string
	CreatedAt     int64
	UserAuthored  bool
}

type chatContext struct {
	ChatID          string
	ChatName        string
	ChatType        string
	CounterpartID   string
	CounterpartName string
	Members         []string
	MemberCount     int
	Anchors         []messageCandidate
	ContextMessages []messageCandidate
}

func (c *Collector) collectChatDomain(ctx context.Context, sessionDir, startISO, endISO string) (DomainResult, ChatSummary, string, DomainDigestStats, error) {
	userOpenID, err := loadSessionUserOpenID(sessionDir)
	if err != nil {
		return DomainResult{
			Domain:  "chat",
			Command: "resolve session user open id",
			Error:   err.Error(),
		}, ChatSummary{}, "", DomainDigestStats{}, nil
	}

	args := []string{
		"im", "+messages-search",
		"--as", "user",
		"--sender", userOpenID,
		"--start", startISO,
		"--end", endISO,
		"--page-size", "50",
		"--page-all",
		"--format", "json",
	}
	out, err := c.runner.Run(ctx, args)
	result := DomainResult{
		Domain:  "chat",
		Command: strings.Join(args, " "),
		Stdout:  out.Stdout,
		Stderr:  out.Stderr,
		Error:   errorString(err),
	}
	if err != nil {
		return result, ChatSummary{}, "", DomainDigestStats{}, nil
	}
	summary, promptData, stats := c.buildChatSummary(ctx, userOpenID, startISO, endISO, out.Stdout)
	return result, summary, promptData, stats, nil
}

func (c *Collector) buildChatSummary(ctx context.Context, userOpenID, startISO, endISO, raw string) (ChatSummary, string, DomainDigestStats) {
	allMessages := extractMessageCandidates(raw)
	anchors := filterAnchorMessages(allMessages, userOpenID)
	stats := DomainDigestStats{
		RawChars: len(raw),
	}
	if len(anchors) == 0 {
		return ChatSummary{}, "", stats
	}

	contexts := c.collectChatContexts(ctx, anchors, startISO, endISO, userOpenID)
	if len(contexts) == 0 {
		return ChatSummary{}, "", stats
	}
	contexts = c.enrichChatContexts(ctx, contexts, userOpenID)
	summary := summarizeChatContexts(contexts, userOpenID)
	promptData, beforeCount, afterCount := buildChatPromptJSON(contexts)
	stats.ItemsBefore = beforeCount
	stats.ItemsAfter = afterCount
	stats.FilteredItems = max(0, stats.ItemsBefore-stats.ItemsAfter)
	stats.FetchedItems = len(contexts)
	return summary, promptData, stats
}

func filterAnchorMessages(messages []messageCandidate, userOpenID string) []messageCandidate {
	filtered := make([]messageCandidate, 0, len(messages))
	for _, msg := range uniqueMessageCandidates(messages) {
		if strings.TrimSpace(msg.ChatID) == "" {
			continue
		}
		if msg.SenderID != userOpenID && !msg.UserAuthored {
			continue
		}
		if !isP2PChat(msg.ChatType) && isNoisyGroupChat(msg.ChatName) {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func (c *Collector) collectChatContexts(ctx context.Context, anchors []messageCandidate, startISO, endISO, userOpenID string) []chatContext {
	byChat := map[string]*chatContext{}
	for _, anchor := range anchors {
		key := strings.TrimSpace(anchor.ChatID)
		if key == "" {
			continue
		}
		entry := byChat[key]
		if entry == nil {
			entry = &chatContext{
				ChatID:        anchor.ChatID,
				ChatName:      anchor.ChatName,
				ChatType:      anchor.ChatType,
				CounterpartID: anchor.CounterpartID,
			}
			byChat[key] = entry
		}
		if entry.CounterpartID == "" {
			entry.CounterpartID = anchor.CounterpartID
		}
		entry.Anchors = append(entry.Anchors, anchor)
	}

	baseContexts := make([]chatContext, 0, len(byChat))
	for _, entry := range byChat {
		baseContexts = append(baseContexts, *entry)
	}
	contexts := runBoundedOrdered(ctx, baseContexts, chatHistoryConcurrency, func(ctx context.Context, entry chatContext) chatContext {
		history := c.fetchChatHistory(ctx, entry.ChatID, startISO, endISO)
		contextMessages := selectChatContextMessages(history, entry.Anchors, userOpenID, entry.ChatType)
		if len(contextMessages) == 0 {
			contextMessages = uniqueMessageCandidates(entry.Anchors)
		}
		entry.ContextMessages = contextMessages
		return entry
	})

	sort.Slice(contexts, func(i, j int) bool {
		if len(contexts[i].Anchors) == len(contexts[j].Anchors) {
			return contexts[i].ChatID < contexts[j].ChatID
		}
		return len(contexts[i].Anchors) > len(contexts[j].Anchors)
	})
	return contexts
}

func (c *Collector) enrichChatContexts(ctx context.Context, contexts []chatContext, userOpenID string) []chatContext {
	counterpartIDs := make([]string, 0, len(contexts))
	for _, entry := range contexts {
		if isP2PChat(entry.ChatType) && strings.TrimSpace(entry.CounterpartID) != "" {
			counterpartIDs = append(counterpartIDs, entry.CounterpartID)
		}
	}
	counterpartNames := c.resolveUserNames(ctx, uniqueStrings(counterpartIDs))

	return runBoundedOrdered(ctx, contexts, extraFetchConcurrency, func(ctx context.Context, entry chatContext) chatContext {
		if isP2PChat(entry.ChatType) {
			_, inferredName := inferPrimaryCounterpart(entry.ContextMessages, userOpenID, entry.ChatName, entry.CounterpartID)
			entry.CounterpartName = firstNonEmpty(counterpartNames[entry.CounterpartID], inferredName)
			return entry
		}
		entry.Members, entry.MemberCount = c.fetchChatMembers(ctx, entry.ChatID)
		return entry
	})
}

func (c *Collector) fetchChatHistory(ctx context.Context, chatID, startISO, endISO string) []messageCandidate {
	if strings.TrimSpace(chatID) == "" {
		return nil
	}
	baseArgs := []string{
		"im", "+chat-messages-list",
		"--as", "user",
		"--chat-id", chatID,
		"--start", startISO,
		"--end", endISO,
		"--page-size", "50",
		"--format", "json",
	}
	var history []messageCandidate
	pageToken := ""
	for {
		args := append([]string{}, baseArgs...)
		if pageToken != "" {
			args = append(args, "--page-token", pageToken)
		}
		out, err := c.runner.Run(ctx, args)
		if err != nil {
			if len(history) == 0 {
				return nil
			}
			break
		}
		history = append(history, extractMessageCandidates(out.Stdout)...)
		hasMore, nextPageToken := extractChatHistoryPage(out.Stdout)
		if !hasMore || strings.TrimSpace(nextPageToken) == "" {
			break
		}
		pageToken = nextPageToken
	}
	return uniqueMessageCandidates(history)
}

func extractChatHistoryPage(raw string) (bool, string) {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return false, ""
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return false, ""
	}
	container := root
	if data, ok := root["data"].(map[string]any); ok {
		container = data
	}
	hasMore, _ := container["has_more"].(bool)
	pageToken := stringValue(container, "page_token", "next_page_token")
	return hasMore, pageToken
}

func selectChatContextMessages(history, anchors []messageCandidate, userOpenID, chatType string) []messageCandidate {
	history = uniqueMessageCandidates(history)
	anchors = uniqueMessageCandidates(anchors)
	if len(history) == 0 || len(anchors) == 0 {
		return nil
	}

	marked := map[int]struct{}{}
	groupLabels := map[string]string{}
	for _, anchor := range anchors {
		if idx, ok := findAnchorIndex(history, anchor, userOpenID); ok {
			marked[idx] = struct{}{}
			if isP2PChat(chatType) {
				for _, triggerIdx := range findP2PTriggerIndexes(history, idx, userOpenID) {
					marked[triggerIdx] = struct{}{}
				}
				continue
			}
			triggerIdx, label := findGroupTriggerIndexAndLabel(history, idx, userOpenID)
			if triggerIdx >= 0 {
				marked[triggerIdx] = struct{}{}
			}
			if label != "" {
				groupLabels[chatMessageKey(history[idx])] = label
			}
		}
	}
	if len(marked) == 0 {
		return nil
	}

	out := make([]messageCandidate, 0, len(marked))
	for i, msg := range history {
		if _, ok := marked[i]; ok {
			if label := groupLabels[chatMessageKey(msg)]; label != "" {
				msg.Text = prefixGroupLabel(msg.Text, label)
			}
			out = append(out, msg)
		}
	}
	return uniqueMessageCandidates(out)
}

func findP2PTriggerIndexes(history []messageCandidate, anchorIdx int, userOpenID string) []int {
	trigger := make([]int, 0, p2pTriggerBlockLimit)
	collecting := false
	for i := anchorIdx - 1; i >= 0; i-- {
		msg := history[i]
		if shouldSkipChatContextMessage(msg) {
			continue
		}
		if msg.SenderID == userOpenID {
			if collecting {
				break
			}
			continue
		}
		collecting = true
		trigger = append(trigger, i)
		if len(trigger) >= p2pTriggerBlockLimit {
			break
		}
	}
	reverseInts(trigger)
	return trigger
}

func findGroupTriggerIndexAndLabel(history []messageCandidate, anchorIdx int, userOpenID string) (int, string) {
	triggerIdx := -1
	seen := 0
	for i := anchorIdx - 1; i >= 0 && seen < groupInitiationLookback; i-- {
		msg := history[i]
		if shouldSkipChatContextMessage(msg) {
			continue
		}
		seen++
		if msg.SenderID == userOpenID {
			continue
		}
		triggerIdx = i
		break
	}
	if triggerIdx >= 0 && messageMentionsUser(history[triggerIdx], userOpenID, history) {
		return triggerIdx, "[被@后回复]"
	}
	if triggerIdx >= 0 {
		return triggerIdx, "[回应他人]"
	}
	return -1, "[群发起话题]"
}

func messageMentionsUser(msg messageCandidate, userOpenID string, history []messageCandidate) bool {
	text := strings.TrimSpace(msg.Text)
	if text == "" || !strings.Contains(text, "@") {
		return false
	}
	for _, name := range selfDisplayNames(history, userOpenID) {
		if strings.Contains(text, "@"+name) {
			return true
		}
	}
	return false
}

func selfDisplayNames(history []messageCandidate, userOpenID string) []string {
	names := make([]string, 0, 2)
	for _, msg := range history {
		if msg.SenderID != userOpenID {
			continue
		}
		names = appendUnique(names, msg.SenderName)
	}
	return names
}

func prefixGroupLabel(text, label string) string {
	text = strings.TrimSpace(text)
	label = strings.TrimSpace(label)
	if text == "" || label == "" || strings.HasPrefix(text, label) {
		return text
	}
	return label + " " + text
}

func stripGroupLabelPrefix(text string) string {
	text = strings.TrimSpace(text)
	for _, label := range []string{"[群发起话题]", "[回应他人]", "[被@后回复]"} {
		if strings.HasPrefix(text, label) {
			return strings.TrimSpace(strings.TrimPrefix(text, label))
		}
	}
	return text
}

func findAnchorIndex(history []messageCandidate, anchor messageCandidate, userOpenID string) (int, bool) {
	anchorID := strings.TrimSpace(anchor.MessageID)
	anchorText := strings.TrimSpace(anchor.Text)
	for i, msg := range history {
		if anchorID != "" && anchorID == strings.TrimSpace(msg.MessageID) {
			return i, true
		}
		if anchorText != "" && msg.SenderID == userOpenID && strings.TrimSpace(msg.Text) == anchorText {
			return i, true
		}
	}
	return 0, false
}

func summarizeChatContexts(contexts []chatContext, userOpenID string) ChatSummary {
	coreCounts := map[string]int{}
	peopleCounts := map[string]int{}
	peopleNames := map[string]string{}
	peopleEvidence := map[string][]string{}
	chatCounts := map[string]int{}
	chatNames := map[string]string{}
	chatEvidence := map[string][]string{}

	for _, ctx := range contexts {
		snippets := textSnippets(ctx.ContextMessages, 3)
		if isP2PChat(ctx.ChatType) {
			counterpartID, counterpartName := inferPrimaryCounterpart(ctx.ContextMessages, userOpenID, ctx.ChatName, ctx.ChatID)
			key := firstNonEmpty(counterpartID, counterpartName, ctx.ChatID)
			coreCounts[key] += len(ctx.Anchors)
			peopleCounts[key] += len(ctx.Anchors) * 3
			if counterpartName != "" {
				peopleNames[key] = counterpartName
			}
			peopleEvidence[key] = appendEvidence(peopleEvidence[key], snippets)
			continue
		}

		chatKey := firstNonEmpty(ctx.ChatID, ctx.ChatName)
		if chatKey != "" {
			chatCounts[chatKey] += len(ctx.Anchors)
			if ctx.ChatName != "" {
				chatNames[chatKey] = ctx.ChatName
			}
			chatEvidence[chatKey] = appendEvidence(chatEvidence[chatKey], snippets)
		}
		for _, msg := range ctx.ContextMessages {
			if msg.SenderID == userOpenID {
				continue
			}
			key := firstNonEmpty(msg.SenderID, msg.SenderName)
			if key == "" {
				continue
			}
			peopleCounts[key]++
			if msg.SenderName != "" {
				peopleNames[key] = msg.SenderName
			}
			if text := strings.TrimSpace(msg.Text); text != "" {
				peopleEvidence[key] = appendUnique(peopleEvidence[key], text)
			}
		}
	}

	topCore := topKeys(coreCounts, 3)
	topChats := topKeys(chatCounts, 3)
	coreSet := make(map[string]struct{}, len(topCore))
	for _, key := range topCore {
		coreSet[key] = struct{}{}
	}
	topPeople := make([]string, 0, 3)
	for _, key := range topKeys(peopleCounts, len(peopleCounts)) {
		if _, duplicatedWithCore := coreSet[key]; duplicatedWithCore {
			continue
		}
		topPeople = append(topPeople, key)
		if len(topPeople) >= 3 {
			break
		}
	}

	coreCollaborators := make([]SummaryItem, 0, len(topCore))
	for _, key := range topCore {
		coreCollaborators = append(coreCollaborators, SummaryItem{
			DisplayName: firstNonEmpty(peopleNames[key], key),
			Identifier:  key,
			Summary:     fmt.Sprintf("围绕你的单聊发言形成高密度往返，上下文中共识别到 %d 个发言锚点。", coreCounts[key]),
			Evidence:    mergeEvidence(peopleEvidence[key], nil),
		})
	}

	frequentPeople := make([]SummaryItem, 0, len(topPeople))
	for _, key := range topPeople {
		frequentPeople = append(frequentPeople, SummaryItem{
			DisplayName: firstNonEmpty(peopleNames[key], key),
			Identifier:  key,
			Summary:     fmt.Sprintf("经常出现在你发言前后的上下文中，累计关联 %d 次。", peopleCounts[key]),
			Evidence:    mergeEvidence(peopleEvidence[key], nil),
		})
	}

	frequentChats := make([]SummaryItem, 0, len(topChats))
	for _, key := range topChats {
		frequentChats = append(frequentChats, SummaryItem{
			DisplayName: firstNonEmpty(chatNames[key], key),
			Identifier:  key,
			Summary:     fmt.Sprintf("过滤闲聊群后，该群保留了 %d 个你的发言锚点，属于高频工作群。", chatCounts[key]),
			Evidence:    mergeEvidence(chatEvidence[key], nil),
		})
	}

	relationshipSummary := fmt.Sprintf(
		"最近 30 天的聊天分析以你的发言为锚点，按单聊触发链和群聊话题锚点提取上下文，并在过滤闲聊群后识别出 %d 位核心协作者和 %d 个你实际参与的工作群。",
		len(topCore),
		len(topChats),
	)
	if len(topCore) == 0 && len(topChats) == 0 {
		relationshipSummary = "最近 30 天存在你的聊天发言，但围绕这些发言形成的稳定协作和工作群信号仍然较弱。"
	}

	return ChatSummary{
		RelationshipSummary: relationshipSummary,
		CoreCollaborators:   coreCollaborators,
		FrequentPeople:      frequentPeople,
		FrequentChats:       frequentChats,
	}
}

func inferPrimaryCounterpart(messages []messageCandidate, userOpenID, fallbackName, fallbackID string) (string, string) {
	counts := map[string]int{}
	names := map[string]string{}
	for _, msg := range messages {
		if msg.SenderID == userOpenID {
			continue
		}
		key := firstNonEmpty(msg.SenderID, msg.SenderName)
		if key == "" {
			continue
		}
		counts[key]++
		if msg.SenderName != "" {
			names[key] = msg.SenderName
		}
	}
	top := topKeys(counts, 1)
	if len(top) == 0 {
		return fallbackID, fallbackName
	}
	key := top[0]
	return key, firstNonEmpty(names[key], key)
}

func isNoisyGroupChat(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return false
	}
	for _, keyword := range noisyGroupKeywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func isP2PChat(chatType string) bool {
	return strings.EqualFold(strings.TrimSpace(chatType), "p2p")
}

func textSnippets(messages []messageCandidate, limit int) []string {
	snippets := make([]string, 0, limit)
	for _, msg := range messages {
		if text := strings.TrimSpace(msg.Text); text != "" {
			snippets = appendUnique(snippets, text)
			if len(snippets) >= limit {
				break
			}
		}
	}
	return snippets
}

func appendEvidence(existing, extra []string) []string {
	merged := append([]string{}, existing...)
	for _, item := range extra {
		merged = appendUnique(merged, item)
	}
	return merged
}

func extractMessageCandidates(raw string) []messageCandidate {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	var candidates []messageCandidate
	var walk func(node any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			senderMap, _ := typed["sender"].(map[string]any)
			partnerMap, _ := typed["chat_partner"].(map[string]any)
			candidate := messageCandidate{
				MessageID:     stringValue(typed, "message_id", "id"),
				ChatID:        stringValue(typed, "chat_id", "open_chat_id"),
				ChatName:      stringValue(typed, "chat_name", "chat_display_name", "name"),
				ChatType:      stringValue(typed, "chat_type", "type"),
				CounterpartID: stringValue(partnerMap, "open_id", "user_id"),
				SenderID:      firstNonEmpty(stringValue(typed, "sender_open_id", "sender_id", "open_id"), stringValue(senderMap, "id")),
				SenderName:    firstNonEmpty(stringValue(typed, "sender_name", "display_name", "user_name"), stringValue(senderMap, "name")),
				Text:          stringValue(typed, "text", "content", "body"),
				CreatedAt:     int64Value(typed, "create_time", "created_at", "message_create_time", "timestamp"),
				UserAuthored:  boolValue(typed, "is_sender", "is_from_me", "is_myself"),
			}
			if candidate.MessageID != "" || candidate.ChatID != "" || candidate.ChatName != "" || candidate.SenderID != "" || candidate.SenderName != "" || candidate.Text != "" {
				candidates = append(candidates, candidate)
			}
			for _, value := range typed {
				walk(value)
			}
		case []any:
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(payload)
	return uniqueMessageCandidates(candidates)
}

func uniqueMessageCandidates(messages []messageCandidate) []messageCandidate {
	seen := map[string]struct{}{}
	out := make([]messageCandidate, 0, len(messages))
	for _, msg := range messages {
		key := strings.TrimSpace(msg.MessageID)
		if key == "" {
			key = strings.Join([]string{
				strings.TrimSpace(msg.ChatID),
				strings.TrimSpace(msg.SenderID),
				strings.TrimSpace(msg.Text),
				strconv.FormatInt(msg.CreatedAt, 10),
			}, "|")
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, msg)
	}
	return out
}

func int64Value(item map[string]any, keys ...string) int64 {
	for _, key := range keys {
		raw, ok := item[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case float64:
			return int64(value)
		case int64:
			return value
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func stringValue(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := item[key]; ok {
			if value, ok := raw.(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func boolValue(item map[string]any, keys ...string) bool {
	for _, key := range keys {
		if raw, ok := item[key]; ok {
			if value, ok := raw.(bool); ok {
				return value
			}
		}
	}
	return false
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func mergeEvidence(primary, extra []string) string {
	merged := append([]string{}, primary...)
	for _, item := range extra {
		merged = appendUnique(merged, item)
	}
	if len(merged) == 0 {
		return "未抽取到足够的上下文片段。"
	}
	return strings.Join(takeFirst(merged, 2), " / ")
}

func topKeys(counts map[string]int, limit int) []string {
	type item struct {
		Key   string
		Count int
	}
	items := make([]item, 0, len(counts))
	for key, count := range counts {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		items = append(items, item{Key: key, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})
	out := make([]string, 0, min(limit, len(items)))
	for i := 0; i < len(items) && i < limit; i++ {
		out = append(out, items[i].Key)
	}
	return out
}

func buildChatPromptJSON(contexts []chatContext) (string, int, int) {
	type promptMessage struct {
		SenderName string `json:"sender_name,omitempty"`
		Text       string `json:"text"`
	}
	type promptContext struct {
		ChatName        string          `json:"chat_name,omitempty"`
		ChatType        string          `json:"chat_type,omitempty"`
		CounterpartName string          `json:"counterpart_name,omitempty"`
		Members         []string        `json:"members,omitempty"`
		MemberCount     int             `json:"member_count,omitempty"`
		Messages        []promptMessage `json:"messages"`
	}

	payload := struct {
		Contexts []promptContext `json:"contexts"`
	}{Contexts: make([]promptContext, 0, len(contexts))}

	totalBefore := 0
	totalAfter := 0
	for _, ctx := range contexts {
		contextMessages := uniqueMessageCandidates(ctx.ContextMessages)
		totalBefore += len(contextMessages)
		filtered := make([]promptMessage, 0, len(contextMessages))
		for _, msg := range contextMessages {
			if !shouldKeepChatPromptMessage(msg) {
				continue
			}
			filtered = append(filtered, promptMessage{
				SenderName: strings.TrimSpace(msg.SenderName),
				Text:       strings.TrimSpace(msg.Text),
			})
		}
		totalAfter += len(filtered)
		if len(filtered) == 0 {
			continue
		}
		members := uniqueStringsStable(ctx.Members)
		if ctx.MemberCount > 20 {
			members = takeFirst(members, groupMemberDisplayLimit)
		}
		payload.Contexts = append(payload.Contexts, promptContext{
			ChatName:        strings.TrimSpace(ctx.ChatName),
			ChatType:        strings.TrimSpace(ctx.ChatType),
			CounterpartName: strings.TrimSpace(ctx.CounterpartName),
			Members:         members,
			MemberCount:     ctx.MemberCount,
			Messages:        filtered,
		})
	}
	if len(payload.Contexts) == 0 {
		return "", totalBefore, totalAfter
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", totalBefore, totalAfter
	}
	return string(data), totalBefore, totalAfter
}

func chatAnchorLookup(anchors []messageCandidate) (map[string]struct{}, []string) {
	lookup := make(map[string]struct{}, len(anchors))
	ids := make([]string, 0, len(anchors))
	for _, anchor := range uniqueMessageCandidates(anchors) {
		key := chatMessageKey(anchor)
		if key == "" {
			continue
		}
		lookup[key] = struct{}{}
		if id := strings.TrimSpace(anchor.MessageID); id != "" {
			ids = append(ids, id)
		}
	}
	return lookup, ids
}

func shouldKeepChatPromptMessage(msg messageCandidate) bool {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return false
	}
	semanticText := stripGroupLabelPrefix(text)
	if isStandaloneURL(semanticText) {
		return false
	}
	if looksLikeLowValueChatMessage(semanticText) {
		return false
	}
	return true
}

func shouldSkipChatContextMessage(msg messageCandidate) bool {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return true
	}
	if isStandaloneURL(text) {
		return true
	}
	return looksLikeLowValueChatMessage(text)
}

func (c *Collector) resolveUserNames(ctx context.Context, userIDs []string) map[string]string {
	type lookupResult struct {
		ID   string
		Name string
	}
	results := runBoundedOrdered(ctx, userIDs, extraFetchConcurrency, func(ctx context.Context, userID string) lookupResult {
		args := []string{
			"contact", "+get-user",
			"--as", "user",
			"--user-id", userID,
			"--user-id-type", "open_id",
			"--format", "json",
		}
		out, err := c.runner.Run(ctx, args)
		if err != nil {
			return lookupResult{ID: userID}
		}
		return lookupResult{ID: userID, Name: extractContactUserName(out.Stdout)}
	})
	names := make(map[string]string, len(results))
	for _, result := range results {
		if strings.TrimSpace(result.ID) == "" || strings.TrimSpace(result.Name) == "" {
			continue
		}
		names[result.ID] = result.Name
	}
	return names
}

func (c *Collector) fetchChatMembers(ctx context.Context, chatID string) ([]string, int) {
	if strings.TrimSpace(chatID) == "" {
		return nil, 0
	}
	params, _ := json.Marshal(map[string]any{
		"chat_id":        chatID,
		"member_id_type": "open_id",
	})
	args := []string{
		"im", "chat.members", "get",
		"--as", "user",
		"--params", string(params),
		"--page-all",
		"--format", "json",
	}
	out, err := c.runner.Run(ctx, args)
	if err != nil {
		return nil, 0
	}
	members, total, unresolved := extractChatMemberInfos(out.Stdout)
	if len(unresolved) > 0 {
		resolved := c.resolveUserNames(ctx, unresolved)
		for _, userID := range unresolved {
			if name := strings.TrimSpace(resolved[userID]); name != "" {
				members = append(members, name)
			}
		}
	}
	members = uniqueStrings(members)
	if total == 0 {
		total = len(members)
	}
	return members, total
}

func extractContactUserName(raw string) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	if data, ok := root["data"].(map[string]any); ok {
		if user, ok := data["user"].(map[string]any); ok {
			return stringValue(user, "name")
		}
	}
	return ""
}

func extractChatMemberInfos(raw string) ([]string, int, []string) {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, 0, nil
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, 0, nil
	}
	container := root
	if data, ok := root["data"].(map[string]any); ok {
		container = data
	}
	items, ok := container["items"].([]any)
	if !ok {
		return nil, intValue(container["member_total"]), nil
	}
	names := make([]string, 0, len(items))
	unresolved := make([]string, 0)
	for _, item := range items {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if name := stringValue(asMap, "name"); name != "" {
			names = append(names, name)
			continue
		}
		if memberID := stringValue(asMap, "member_id"); memberID != "" {
			unresolved = append(unresolved, memberID)
		}
	}
	return names, intValue(container["member_total"]), uniqueStrings(unresolved)
}

func intValue(raw any) int {
	switch value := raw.(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func uniqueStringsStable(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func reverseInts(values []int) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}

func chatMessageKey(msg messageCandidate) string {
	if messageID := strings.TrimSpace(msg.MessageID); messageID != "" {
		return messageID
	}
	return strings.TrimSpace(msg.SenderID) + "|" + strings.TrimSpace(msg.Text) + "|" + strconv.FormatInt(msg.CreatedAt, 10)
}

func isStandaloneURL(text string) bool {
	text = strings.TrimSpace(text)
	return (strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")) && !strings.Contains(text, " ")
}

func looksLikeLowValueChatMessage(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	if isHighSignalChatMessage(text) {
		return false
	}
	normalized := strings.ToLower(strings.ReplaceAll(text, " ", ""))
	switch normalized {
	case "好", "好的", "收到", "收到啦", "收到收到", "ok", "okok", "okay", "嗯", "嗯嗯", "行", "好滴", "👌", "👍", "哈哈", "hhh", "thx", "thanks", "收到👌", "好的👌":
		return true
	case "hi", "hello", "早", "早上好", "午安", "晚安":
		return true
	case "[图片]", "[image]", "image", "[表情]", "表情", "[sticker]", "sticker", "贴纸", "动画表情":
		return true
	}
	if isPureEmojiLikeMessage(text) {
		return true
	}
	if len([]rune(text)) <= 4 && !strings.ContainsAny(text, "0123456789@#`") {
		return true
	}
	return false
}

func isHighSignalChatMessage(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if len([]rune(text)) >= 15 {
		return true
	}
	if strings.Contains(text, "http://") || strings.Contains(text, "https://") {
		return true
	}
	if strings.ContainsAny(text, "@`#") {
		return true
	}
	if strings.ContainsAny(text, "？?") {
		return true
	}
	if strings.ContainsAny(text, "0123456789") {
		return true
	}
	for _, marker := range []string{"- ", "* ", "1.", "2.", "3.", "TODO", "todo", "接口", "方案", "测试", "排期", "结论", "风险", "上线", "推进", "同步", "评审", "review", "fix", "ship", "bug", "问题", "确认", "看了吗", "帮忙"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func isPureEmojiLikeMessage(text string) bool {
	hasSignal := false
	for _, r := range text {
		switch {
		case r >= '0' && r <= '9':
			return false
		case r >= 'a' && r <= 'z':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		case r >= 0x4E00 && r <= 0x9FFF:
			return false
		case strings.ContainsRune("@#?_!,.，。！？-+:/", r):
			return false
		case r == ' ' || r == '\t' || r == '\n':
			continue
		default:
			hasSignal = true
		}
	}
	return hasSignal
}
