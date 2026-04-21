package collector

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"feishu-personality-agent/internal/persona"
	"feishu-personality-agent/internal/sandbox"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (f *fakeRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string(nil), args...))
	f.mu.Unlock()
	return sandbox.Output{Stdout: `{"ok":true}`}, nil
}

func (f *fakeRunner) Calls() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneCalls(f.calls)
}

func TestCollectWritesPrivateRawLogsForCoreDomains(t *testing.T) {
	dir := t.TempDir()
	writeSessionUserConfig(t, dir, "ou_self", "Self User")
	runner := &fakeRunner{}
	c := New(runner)

	bundle, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Domains) != 6 {
		t.Fatalf("domains = %d", len(bundle.Domains))
	}
	for _, domain := range []string{"chat", "docs", "calendar", "task", "mail", "vc"} {
		path := filepath.Join(dir, "logs", "raw-"+domain+".jsonl")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing raw log %s: %v", domain, err)
		}
	}
	if calls := runner.Calls(); len(calls) != 7 {
		t.Fatalf("calls = %d", len(calls))
	}
}

type scriptedRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (s *scriptedRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	s.mu.Lock()
	s.calls = append(s.calls, append([]string(nil), args...))
	s.mu.Unlock()
	switch {
	case len(args) >= 2 && args[0] == "im" && args[1] == "+messages-search":
		return sandbox.Output{Stdout: `{"items":[
			{"message_id":"m_noise_1","chat_id":"oc_noise_1","chat_name":"周末吃饭闲聊群","chat_type":"group","sender_open_id":"ou_self","sender_name":"Self User","text":"俺也去吗","is_sender":true,"create_time":"100"},
			{"message_id":"m_group_1","chat_id":"oc_group_1","chat_name":"跨团队项目群","chat_type":"group","sender_open_id":"ou_self","sender_name":"Self User","text":"我来同步接口进度","is_sender":true,"create_time":"200"},
			{"message_id":"m_p2p_1","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","chat_partner":{"open_id":"ou_peer_1"},"sender_open_id":"ou_self","sender_name":"Self User","text":"我先整理方案","is_sender":true,"create_time":"300"},
			{"message_id":"m_p2p_2","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","chat_partner":{"open_id":"ou_peer_1"},"sender_open_id":"ou_self","sender_name":"Self User","text":"明天过一下细节","is_sender":true,"create_time":"400"}
		]}`}, nil
	case len(args) >= 2 && args[0] == "im" && args[1] == "+chat-messages-list":
		if hasArgValue(args, "--chat-id", "oc_group_1") {
			return sandbox.Output{Stdout: `{"ok":true,"data":{"has_more":false,"page_token":"","messages":[
				{"message_id":"g0","chat_id":"oc_group_1","chat_name":"跨团队项目群","chat_type":"group","sender_open_id":"ou_peer_2","sender_name":"小王","text":"今天先过一下 blocker","create_time":"190"},
				{"message_id":"m_group_1","chat_id":"oc_group_1","chat_name":"跨团队项目群","chat_type":"group","sender_open_id":"ou_self","sender_name":"Self User","text":"我来同步接口进度","create_time":"200"},
				{"message_id":"g2","chat_id":"oc_group_1","chat_name":"跨团队项目群","chat_type":"group","sender_open_id":"ou_peer_2","sender_name":"小王","text":"收到，我补充测试计划","create_time":"210"}
			]}}`}, nil
		}
		return sandbox.Output{Stdout: `{"ok":true,"data":{"has_more":false,"page_token":"","messages":[
			{"message_id":"p0","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_peer_1","sender_name":"小李","text":"我先看一下背景","create_time":"290"},
			{"message_id":"m_p2p_1","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_self","sender_name":"Self User","text":"我先整理方案","create_time":"300"},
			{"message_id":"p2","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_peer_1","sender_name":"小李","text":"你先看下接口定义，我补用例","create_time":"310"},
			{"message_id":"p3","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_self","sender_name":"Self User","text":"明天过一下细节","create_time":"400"},
			{"message_id":"p4","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_peer_1","sender_name":"小李","text":"好，我下午给你确认","create_time":"410"}
		]}}`}, nil
	case len(args) >= 2 && args[0] == "im" && args[1] == "+chat-search":
		return sandbox.Output{Stdout: `{"items":[{"chat_id":"oc_group_1","name":"跨团队项目群"}]}`}, nil
	case len(args) >= 3 && args[0] == "im" && args[1] == "chat.members" && args[2] == "get":
		return sandbox.Output{Stdout: `{"data":{"items":[
			{"member_id":"ou_self","name":"Self User"},
			{"member_id":"ou_peer_2","name":"小王"},
			{"member_id":"ou_peer_3","name":"小赵"}
		],"member_total":3}}`}, nil
	case len(args) >= 2 && args[0] == "contact" && args[1] == "+get-user":
		if hasArgValue(args, "--user-id", "ou_peer_1") {
			return sandbox.Output{Stdout: `{"data":{"user":{"name":"刘岳林","user_id":"ou_peer_1"}}}`}, nil
		}
		return sandbox.Output{Stdout: `{"data":{"user":{"name":"未知成员"}}}`}, nil
	case len(args) >= 2 && args[0] == "docs" && args[1] == "+search":
		if strings.Contains(strings.Join(args, " "), `"creator_ids":["ou_self"]`) {
			return sandbox.Output{Stdout: `{"data":{"results":[
				{"entity_type":"WIKI","result_meta":{"token":"alpha","url":"https://example.feishu.cn/wiki/alpha"},"title_highlighted":"个人方案复盘"},
				{"entity_type":"DOC","result_meta":{"token":"beta","url":"https://example.feishu.cn/docx/beta"},"title_highlighted":"个人周报"},
				{"entity_type":"DOC","result_meta":{"token":"gamma","url":"https://example.feishu.cn/docx/gamma"},"title_highlighted":"技术进展总结"},
				{"entity_type":"DOC","result_meta":{"token":"theta","url":"https://example.feishu.cn/docx/theta"},"title_highlighted":"项目推进节奏"},
				{"entity_type":"DOC","result_meta":{"token":"iota","url":"https://example.feishu.cn/docx/iota"},"title_highlighted":"评审前检查清单"},
				{"entity_type":"DOC","result_meta":{"token":"kappa","url":"https://example.feishu.cn/docx/kappa"},"title_highlighted":"接口联调记录"},
				{"entity_type":"DOC","result_meta":{"token":"lambda","url":"https://example.feishu.cn/docx/lambda"},"title_highlighted":"风险复盘"},
				{"entity_type":"DOC","result_meta":{"token":"mu","url":"https://example.feishu.cn/docx/mu"},"title_highlighted":"知识沉淀笔记"},
				{"entity_type":"DOC","result_meta":{"token":"nu","url":"https://example.feishu.cn/docx/nu"},"title_highlighted":"路线规划"},
				{"entity_type":"DOC","result_meta":{"token":"xi","url":"https://example.feishu.cn/docx/xi"},"title_highlighted":"技术选型比较"}
			]}}`}, nil
		}
		return sandbox.Output{Stdout: `{"data":{"results":[
			{"entity_type":"DOC","result_meta":{"token":"delta","url":"https://example.feishu.cn/docx/delta"},"title_highlighted":"最近浏览设计文档"},
			{"entity_type":"DOC","result_meta":{"token":"epsilon","url":"https://example.feishu.cn/docx/epsilon"},"title_highlighted":"会议纪要"}
		]}}`}, nil
	case len(args) >= 2 && args[0] == "docs" && args[1] == "+fetch":
		docRef := args[3]
		return sandbox.Output{Stdout: `{"data":{"title":"` + docRef + `","markdown":"# ` + docRef + `\n- 方案约束\n- 风险边界\n- 推进结论"}}`}, nil
	case len(args) >= 2 && args[0] == "vc" && args[1] == "+search":
		return sandbox.Output{Stdout: `{"items":[{"calendar_event_id":"evt-1","title":"技术评审会","organizer":"Self User"},{"calendar_event_id":"evt-2","title":"周会同步","organizer":"Lead"}]}`}, nil
	case len(args) >= 2 && args[0] == "vc" && args[1] == "+notes":
		return sandbox.Output{Stdout: `{"notes":"议题：方案评审；结论：先补压测；行动项：周三前完成"}`}, nil
	case len(args) >= 2 && args[0] == "mail" && args[1] == "+triage":
		return sandbox.Output{Stdout: `{"items":[{"message_id":"mid-1","subject":"方案评审反馈","from":"teammate@example.com"},{"message_id":"mid-2","subject":"GitHub notification","from":"noreply@github.com"}]}`}, nil
	case len(args) >= 2 && args[0] == "mail" && args[1] == "+message":
		if args[3] == "mid-1" {
			return sandbox.Output{Stdout: `{"body":"建议把风险和回滚方案写进文档，再给出明确上线窗口。"}`}, nil
		}
		return sandbox.Output{Stdout: `{"body":"notification digest"}`}, nil
	default:
		return sandbox.Output{Stdout: `{"ok":true}`}, nil
	}
}

func (s *scriptedRunner) Calls() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneCalls(s.calls)
}

func TestCollectFetchesDocBodiesMeetingNotesAndMailBodies(t *testing.T) {
	dir := t.TempDir()
	writeSessionUserConfig(t, dir, "ou_self", "Self User")
	runner := &scriptedRunner{}
	c := New(runner)

	bundle, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Domains) != 18 {
		t.Fatalf("domains = %d", len(bundle.Domains))
	}
	gotDomains := make([]string, 0, len(bundle.Domains))
	for _, domain := range bundle.Domains {
		gotDomains = append(gotDomains, domain.Domain)
	}
	wantDomains := []string{"chat", "docs", "docs_content", "docs_content", "docs_content", "docs_content", "docs_content", "docs_content", "docs_content", "docs_content", "docs_content", "docs_content", "calendar", "task", "mail", "mail_content", "vc", "vc_notes"}
	if strings.Join(gotDomains, ",") != strings.Join(wantDomains, ",") {
		t.Fatalf("domain order = %#v, want %#v", gotDomains, wantDomains)
	}

	var sawDocFetch, sawVCNotes, sawMailMessage bool
	for _, call := range runner.Calls() {
		if len(call) >= 2 && call[0] == "docs" && call[1] == "+fetch" {
			sawDocFetch = true
		}
		if len(call) >= 2 && call[0] == "vc" && call[1] == "+notes" {
			sawVCNotes = true
		}
		if len(call) >= 2 && call[0] == "mail" && call[1] == "+message" {
			sawMailMessage = true
		}
	}
	if !sawDocFetch {
		t.Fatal("expected docs +fetch")
	}
	if !sawVCNotes {
		t.Fatal("expected vc +notes")
	}
	if !sawMailMessage {
		t.Fatal("expected mail +message")
	}
	if bundle.ChatSummary.RelationshipSummary == "" {
		t.Fatalf("chat summary = %#v", bundle.ChatSummary)
	}
	if len(bundle.ChatSummary.FrequentChats) == 0 {
		t.Fatalf("chat summary missing frequent targets: %#v", bundle.ChatSummary)
	}
	chatRaw := bundle.Domains[0].Stdout
	for _, want := range []string{`"counterpart_name": "刘岳林"`, `"members": [`, `"sender_name": "Self User"`, `"text": "我先整理方案"`} {
		if !strings.Contains(chatRaw, want) {
			t.Fatalf("chat prompt missing %q: %s", want, chatRaw)
		}
	}
	for _, unwanted := range []string{`"message_id"`, `"sender_id"`, `"chat_id"`, `"created_at"`, `"anchor_message_ids"`, `"user_authored"`} {
		if strings.Contains(chatRaw, unwanted) {
			t.Fatalf("chat prompt should not contain %s: %s", unwanted, chatRaw)
		}
	}
	if len(bundle.DocsSummary.PersonalDocs) != 10 || len(bundle.DocsSummary.SelectedDocs) != 12 {
		t.Fatalf("docs summary = %#v", bundle.DocsSummary)
	}
	docsArgs, ok := findCall(runner.Calls(), "docs", "+search")
	if !ok {
		t.Fatal("docs search call not found")
	}
	joined := strings.Join(docsArgs, " ")
	if !strings.Contains(joined, `"creator_ids":["ou_self"]`) {
		t.Fatalf("docs personal search should use session user creator_ids: %#v", docsArgs)
	}
	if strings.Contains(joined, `"owners":["me"]`) {
		t.Fatalf("docs personal search should not use owners=[\"me\"]: %#v", docsArgs)
	}
}

func TestCollectOmitsEmptyChatQueryArgument(t *testing.T) {
	dir := t.TempDir()
	writeSessionUserConfig(t, dir, "ou_self", "Self User")
	runner := &fakeRunner{}
	c := New(runner)

	if _, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))); err != nil {
		t.Fatal(err)
	}
	calls := runner.Calls()
	if len(calls) == 0 {
		t.Fatal("no collector calls recorded")
	}
	chatArgs, ok := findCall(calls, "im", "+messages-search")
	if !ok {
		t.Fatalf("chat search call not found: %#v", calls)
	}
	for i, arg := range chatArgs {
		if arg == "" {
			t.Fatalf("chat arg %d is empty: %#v", i, chatArgs)
		}
		if arg == "--query" {
			t.Fatalf("chat args include --query for blank search: %#v", chatArgs)
		}
	}
}

func TestCollectUsesUserPaginatedChatSearchWithoutPageLimit(t *testing.T) {
	dir := t.TempDir()
	writeSessionUserConfig(t, dir, "ou_self", "Self User")
	runner := &fakeRunner{}
	c := New(runner)

	if _, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))); err != nil {
		t.Fatal(err)
	}
	calls := runner.Calls()
	if len(calls) == 0 {
		t.Fatal("no collector calls recorded")
	}

	chatArgs, ok := findCall(calls, "im", "+messages-search")
	if !ok {
		t.Fatalf("chat search call not found: %#v", calls)
	}
	if !containsArg(chatArgs, "--as") || !containsArg(chatArgs, "user") {
		t.Fatalf("chat args should include explicit user identity: %#v", chatArgs)
	}
	if !containsArg(chatArgs, "--page-all") {
		t.Fatalf("chat args should enable auto pagination: %#v", chatArgs)
	}
	if containsArg(chatArgs, "--page-limit") {
		t.Fatalf("chat args should not include page-limit: %#v", chatArgs)
	}
	if !hasArgValue(chatArgs, "--sender", "ou_self") {
		t.Fatalf("chat args should include session user open id: %#v", chatArgs)
	}
}

func TestLoadSessionUserOpenID(t *testing.T) {
	dir := t.TempDir()
	writeSessionUserConfig(t, dir, "ou_self", "Self User")

	got, err := loadSessionUserOpenID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ou_self" {
		t.Fatalf("open id = %q", got)
	}
}

func TestSelectChatContextMessagesForP2PKeepsLatestTriggerBlock(t *testing.T) {
	history := []messageCandidate{
		{MessageID: "m0", SenderID: "ou_peer", SenderName: "小李", Text: "哈哈"},
		{MessageID: "m1", SenderID: "ou_peer", SenderName: "小李", Text: "那个bug你看了吗"},
		{MessageID: "m2", SenderID: "ou_self", SenderName: "Self User", Text: "嗯"},
		{MessageID: "m3", SenderID: "ou_peer", SenderName: "小李", Text: "你review一下"},
		{MessageID: "m4", SenderID: "ou_self", SenderName: "Self User", Text: "看了，是配置问题，我改了个PR"},
		{MessageID: "m5", SenderID: "ou_self", SenderName: "Self User", Text: "你review一下"},
	}
	anchors := []messageCandidate{
		{MessageID: "m4", SenderID: "ou_self", Text: "看了，是配置问题，我改了个PR"},
		{MessageID: "m5", SenderID: "ou_self", Text: "你review一下"},
	}

	got := selectChatContextMessages(history, anchors, "ou_self", "p2p")
	if len(got) != 4 {
		t.Fatalf("context size = %d, want 4", len(got))
	}
	if got[0].MessageID != "m1" || got[1].MessageID != "m3" {
		t.Fatalf("p2p trigger block = %#v", got)
	}
	if got[2].MessageID != "m4" || got[3].MessageID != "m5" {
		t.Fatalf("p2p anchors = %#v", got)
	}
}

func TestSelectChatContextMessagesForGroupAddsInlineRoleMarkers(t *testing.T) {
	history := []messageCandidate{
		{MessageID: "g0", SenderID: "ou_peer_1", SenderName: "小王", Text: "@Self User 那个接口你看了吗"},
		{MessageID: "g1", SenderID: "ou_self", SenderName: "Self User", Text: "我看了，晚点补个结论"},
		{MessageID: "g2", SenderID: "ou_self", SenderName: "Self User", Text: "我先补个背景"},
		{MessageID: "g3", SenderID: "ou_self", SenderName: "Self User", Text: "再同步一下范围"},
		{MessageID: "g4", SenderID: "ou_self", SenderName: "Self User", Text: "我先起个话题"},
	}
	anchors := []messageCandidate{
		{MessageID: "g1", SenderID: "ou_self", Text: "我看了，晚点补个结论"},
		{MessageID: "g4", SenderID: "ou_self", Text: "我先起个话题"},
	}

	got := selectChatContextMessages(history, anchors, "ou_self", "group")
	if len(got) != 3 {
		t.Fatalf("group context size = %d, want 3", len(got))
	}
	if got[0].MessageID != "g0" {
		t.Fatalf("group anchor = %#v", got)
	}
	if got[1].Text != "[被@后回复] 我看了，晚点补个结论" {
		t.Fatalf("mention reply label = %#v", got[1])
	}
	if got[2].Text != "[群发起话题] 我先起个话题" {
		t.Fatalf("topic-init label = %#v", got[2])
	}
}

type paginatedChatHistoryRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (p *paginatedChatHistoryRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	p.mu.Lock()
	p.calls = append(p.calls, append([]string(nil), args...))
	p.mu.Unlock()
	if len(args) >= 2 && args[0] == "im" && args[1] == "+chat-messages-list" {
		if hasArgValue(args, "--page-token", "token-2") {
			return sandbox.Output{Stdout: `{"ok":true,"data":{"has_more":false,"page_token":"","messages":[{"message_id":"m2","chat_id":"oc_chat_1","chat_type":"p2p","sender_open_id":"ou_peer","sender_name":"小李","text":"第二页","create_time":"200"}]}}`}, nil
		}
		return sandbox.Output{Stdout: `{"ok":true,"data":{"has_more":true,"page_token":"token-2","messages":[{"message_id":"m1","chat_id":"oc_chat_1","chat_type":"p2p","sender_open_id":"ou_self","sender_name":"Self User","text":"第一页","create_time":"100"}]}}`}, nil
	}
	return sandbox.Output{Stdout: `{"ok":true}`}, nil
}

func (p *paginatedChatHistoryRunner) Calls() [][]string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneCalls(p.calls)
}

type paginatedChatHistoryFailRunner struct{}

func (p *paginatedChatHistoryFailRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	if len(args) >= 2 && args[0] == "im" && args[1] == "+chat-messages-list" {
		if hasArgValue(args, "--page-token", "token-2") {
			return sandbox.Output{}, errors.New("page 2 failed")
		}
		return sandbox.Output{Stdout: `{"ok":true,"data":{"has_more":true,"page_token":"token-2","messages":[{"message_id":"m1","chat_id":"oc_chat_1","chat_type":"p2p","sender_open_id":"ou_self","sender_name":"Self User","text":"第一页","create_time":"100"}]}}`}, nil
	}
	return sandbox.Output{Stdout: `{"ok":true}`}, nil
}

func TestFetchChatHistoryPaginatesUntilComplete(t *testing.T) {
	runner := &paginatedChatHistoryRunner{}
	c := New(runner)

	history := c.fetchChatHistory(context.Background(), "oc_chat_1", "2026-03-01T00:00:00+08:00", "2026-04-01T00:00:00+08:00")
	if len(history) != 2 {
		t.Fatalf("history = %#v", history)
	}
	if history[0].Text != "第一页" || history[1].Text != "第二页" {
		t.Fatalf("history order = %#v", history)
	}
	calls := runner.Calls()
	if len(calls) != 2 {
		t.Fatalf("calls = %#v", calls)
	}
	if containsArg(calls[0], "--page-token") {
		t.Fatalf("first page should not include page token: %#v", calls[0])
	}
	if !hasArgValue(calls[1], "--page-token", "token-2") {
		t.Fatalf("second page should use returned page token: %#v", calls[1])
	}
	if containsArg(calls[0], "--page-all") || containsArg(calls[1], "--page-all") {
		t.Fatalf("chat history paging should not use --page-all: %#v", calls)
	}
}

func TestFetchChatHistoryKeepsCollectedPagesWhenLaterPageFails(t *testing.T) {
	c := New(&paginatedChatHistoryFailRunner{})

	history := c.fetchChatHistory(context.Background(), "oc_chat_1", "2026-03-01T00:00:00+08:00", "2026-04-01T00:00:00+08:00")
	if len(history) != 1 {
		t.Fatalf("history = %#v", history)
	}
	if history[0].Text != "第一页" {
		t.Fatalf("history = %#v", history)
	}
}

func TestCollectChatSummaryAnchorsOnUserSpeechAndFiltersNoisyGroups(t *testing.T) {
	dir := t.TempDir()
	writeSessionUserConfig(t, dir, "ou_self", "Self User")
	runner := &scriptedRunner{}
	c := New(runner)

	bundle, err := c.Collect(context.Background(), dir, time.Date(2026, 4, 10, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.ChatSummary.CoreCollaborators) == 0 {
		t.Fatalf("core collaborators = %#v", bundle.ChatSummary)
	}
	if bundle.ChatSummary.CoreCollaborators[0].DisplayName != "小李" {
		t.Fatalf("core collaborator = %#v", bundle.ChatSummary.CoreCollaborators[0])
	}
	if !strings.Contains(bundle.ChatSummary.CoreCollaborators[0].Evidence, "我先整理方案") {
		t.Fatalf("core evidence = %q", bundle.ChatSummary.CoreCollaborators[0].Evidence)
	}
	for _, item := range bundle.ChatSummary.FrequentPeople {
		if item.DisplayName == bundle.ChatSummary.CoreCollaborators[0].DisplayName {
			t.Fatalf("frequent people should exclude core collaborators: %#v", bundle.ChatSummary)
		}
	}
	if len(bundle.ChatSummary.FrequentChats) != 1 || bundle.ChatSummary.FrequentChats[0].DisplayName != "跨团队项目群" {
		t.Fatalf("frequent chats = %#v", bundle.ChatSummary.FrequentChats)
	}
	for _, item := range bundle.ChatSummary.FrequentChats {
		if strings.Contains(item.DisplayName, "闲聊") {
			t.Fatalf("noisy group should be filtered: %#v", bundle.ChatSummary.FrequentChats)
		}
	}
	for _, call := range runner.Calls() {
		if len(call) >= 2 && call[0] == "im" && call[1] == "+chat-messages-list" && hasArgValue(call, "--chat-id", "oc_noise_1") {
			t.Fatalf("noisy group history should not be fetched: %#v", runner.Calls())
		}
		if len(call) >= 2 && call[0] == "im" && call[1] == "+chat-messages-list" && containsArg(call, "--page-all") {
			t.Fatalf("chat history fetch should use explicit page tokens instead of --page-all: %#v", call)
		}
	}
}

func TestExtractDocSummariesIgnoresNestedNonDocNames(t *testing.T) {
	raw := `{
		"data": {"results": [
			{
				"title_highlighted": "设计文档",
				"result_meta": {"url": "https://example.feishu.cn/docx/alpha", "token":"alpha"},
				"owner": {"name": "张三"},
				"creator": {"name": "李四"}
			},
			{
				"title_highlighted": "方案复盘",
				"result_meta": {"token":"doc_beta"},
				"modifier": {"name": "王五"}
			}
		]}
	}`

	got := extractDocSummaries(raw, "recent_open")
	if len(got) != 2 {
		t.Fatalf("doc summaries = %#v", got)
	}
	if got[0].Title != "设计文档" || got[0].Identifier != "alpha" {
		t.Fatalf("first doc summary = %#v", got[0])
	}
	if got[1].Title != "方案复盘" || got[1].Identifier != "doc_beta" {
		t.Fatalf("second doc summary = %#v", got[1])
	}
	for _, item := range got {
		if item.Title == "张三" || item.Title == "李四" || item.Title == "王五" {
			t.Fatalf("nested non-doc names should not be extracted: %#v", got)
		}
	}
}

func TestExtractDocSummariesParsesDataResultsAndResultMeta(t *testing.T) {
	raw := `{
		"data": {
			"results": [
				{
					"title_highlighted": "第十三周",
					"result_meta": {
						"token": "PG7mwODeFiOISIkBoiZco3CHnTe",
						"url": "https://bytedance.larkoffice.com/wiki/PG7mwODeFiOISIkBoiZco3CHnTe"
					}
				}
			]
		}
	}`

	got := extractDocSummaries(raw, "created_by_self")
	if len(got) != 1 {
		t.Fatalf("doc summaries = %#v", got)
	}
	if got[0].Title != "第十三周" {
		t.Fatalf("title = %#v", got[0])
	}
	if got[0].URL != "https://bytedance.larkoffice.com/wiki/PG7mwODeFiOISIkBoiZco3CHnTe" {
		t.Fatalf("url = %#v", got[0])
	}
	if got[0].Identifier != "PG7mwODeFiOISIkBoiZco3CHnTe" {
		t.Fatalf("identifier = %#v", got[0])
	}
}

type concurrentDocsRunner struct {
	mu            sync.Mutex
	calls         [][]string
	searchStarted int
	releaseSearch chan struct{}
}

func newConcurrentDocsRunner() *concurrentDocsRunner {
	return &concurrentDocsRunner{releaseSearch: make(chan struct{})}
}

func (r *concurrentDocsRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	r.mu.Lock()
	r.calls = append(r.calls, append([]string(nil), args...))
	r.mu.Unlock()
	if len(args) >= 2 && args[0] == "docs" && args[1] == "+search" {
		r.mu.Lock()
		r.searchStarted++
		started := r.searchStarted
		release := r.releaseSearch
		if started == 2 {
			close(release)
		}
		r.mu.Unlock()
		select {
		case <-release:
		case <-time.After(100 * time.Millisecond):
			return sandbox.Output{}, errors.New("docs search did not overlap")
		case <-ctx.Done():
			return sandbox.Output{}, ctx.Err()
		}
		if strings.Contains(strings.Join(args, " "), `"creator_ids":["ou_self"]`) {
			return sandbox.Output{Stdout: `{"data":{"results":[
				{"entity_type":"WIKI","result_meta":{"token":"alpha","url":"https://example.feishu.cn/wiki/alpha"},"title_highlighted":"个人方案复盘"},
				{"entity_type":"DOC","result_meta":{"token":"beta","url":"https://example.feishu.cn/docx/beta"},"title_highlighted":"个人周报"},
				{"entity_type":"DOC","result_meta":{"token":"gamma","url":"https://example.feishu.cn/docx/gamma"},"title_highlighted":"技术进展总结"},
				{"entity_type":"DOC","result_meta":{"token":"theta","url":"https://example.feishu.cn/docx/theta"},"title_highlighted":"项目推进节奏"},
				{"entity_type":"DOC","result_meta":{"token":"iota","url":"https://example.feishu.cn/docx/iota"},"title_highlighted":"评审前检查清单"},
				{"entity_type":"DOC","result_meta":{"token":"kappa","url":"https://example.feishu.cn/docx/kappa"},"title_highlighted":"接口联调记录"},
				{"entity_type":"DOC","result_meta":{"token":"lambda","url":"https://example.feishu.cn/docx/lambda"},"title_highlighted":"风险复盘"},
				{"entity_type":"DOC","result_meta":{"token":"mu","url":"https://example.feishu.cn/docx/mu"},"title_highlighted":"知识沉淀笔记"},
				{"entity_type":"DOC","result_meta":{"token":"nu","url":"https://example.feishu.cn/docx/nu"},"title_highlighted":"路线规划"},
				{"entity_type":"DOC","result_meta":{"token":"xi","url":"https://example.feishu.cn/docx/xi"},"title_highlighted":"技术选型比较"}
			]}}`}, nil
		}
		return sandbox.Output{Stdout: `{"data":{"results":[
			{"entity_type":"DOC","result_meta":{"token":"delta","url":"https://example.feishu.cn/docx/delta"},"title_highlighted":"最近浏览设计文档"},
			{"entity_type":"DOC","result_meta":{"token":"epsilon","url":"https://example.feishu.cn/docx/epsilon"},"title_highlighted":"会议纪要"}
		]}}`}, nil
	}
	if len(args) >= 2 && args[0] == "docs" && args[1] == "+fetch" {
		return sandbox.Output{Stdout: `{"content":"doc body"}`}, nil
	}
	return sandbox.Output{Stdout: `{"ok":true}`}, nil
}

func TestCollectDocsDomainRunsSearchesConcurrently(t *testing.T) {
	dir := t.TempDir()
	writeSessionUserConfig(t, dir, "ou_self", "Self User")
	runner := newConcurrentDocsRunner()
	c := New(runner)

	result, extras, summary, err := c.collectDocsDomain(context.Background(), dir, "2026-03-10T12:00:00+08:00", "2026-04-10T12:00:00+08:00")
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != "" {
		t.Fatalf("docs result error = %q", result.Error)
	}
	if len(summary.PersonalDocs) != 10 || len(summary.RecentDocs) != 2 || len(summary.SelectedDocs) != 12 {
		t.Fatalf("docs summary = %#v", summary)
	}
	if len(extras) != 10 {
		t.Fatalf("docs extras = %#v", extras)
	}
}

type concurrentHistoryRunner struct {
	mu             sync.Mutex
	calls          [][]string
	historyStarted int
	releaseHistory chan struct{}
}

func newConcurrentHistoryRunner() *concurrentHistoryRunner {
	return &concurrentHistoryRunner{releaseHistory: make(chan struct{})}
}

func (r *concurrentHistoryRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	r.mu.Lock()
	r.calls = append(r.calls, append([]string(nil), args...))
	r.mu.Unlock()
	if len(args) >= 2 && args[0] == "im" && args[1] == "+chat-messages-list" {
		r.mu.Lock()
		r.historyStarted++
		started := r.historyStarted
		release := r.releaseHistory
		if started == 2 {
			close(release)
		}
		r.mu.Unlock()
		select {
		case <-release:
		case <-time.After(100 * time.Millisecond):
			return sandbox.Output{}, errors.New("chat history did not overlap")
		case <-ctx.Done():
			return sandbox.Output{}, ctx.Err()
		}
		chatID := ""
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "--chat-id" {
				chatID = args[i+1]
				break
			}
		}
		switch chatID {
		case "oc_group_1":
			return sandbox.Output{Stdout: `{"items":[
				{"message_id":"g0","chat_id":"oc_group_1","chat_name":"跨团队项目群","chat_type":"group","sender_open_id":"ou_peer_2","sender_name":"小王","text":"收到，我补充测试计划","create_time":"210"},
				{"message_id":"g1","chat_id":"oc_group_1","chat_name":"跨团队项目群","chat_type":"group","sender_open_id":"ou_self","sender_name":"Self User","text":"我来同步接口进度","create_time":"220"},
				{"message_id":"g2","chat_id":"oc_group_1","chat_name":"跨团队项目群","chat_type":"group","sender_open_id":"ou_peer_2","sender_name":"小王","text":"今天先过一下 blocker","create_time":"230"}
			]}`}, nil
		case "oc_p2p_1":
			return sandbox.Output{Stdout: `{"items":[
				{"message_id":"p0","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_peer_1","sender_name":"小李","text":"我先看一下背景","create_time":"290"},
				{"message_id":"p1","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_self","sender_name":"Self User","text":"我先整理方案","create_time":"300"},
				{"message_id":"p2","chat_id":"oc_p2p_1","chat_name":"与小李单聊","chat_type":"p2p","sender_open_id":"ou_peer_1","sender_name":"小李","text":"你先看下接口定义，我补用例","create_time":"310"}
			]}`}, nil
		}
	}
	return sandbox.Output{Stdout: `{"ok":true}`}, nil
}

func TestCollectChatContextsFetchesHistoriesConcurrently(t *testing.T) {
	runner := newConcurrentHistoryRunner()
	c := New(runner)
	anchors := []messageCandidate{
		{MessageID: "m_group_1", ChatID: "oc_group_1", ChatName: "跨团队项目群", ChatType: "group", SenderID: "ou_self", Text: "我来同步接口进度"},
		{MessageID: "m_p2p_1", ChatID: "oc_p2p_1", ChatName: "与小李单聊", ChatType: "p2p", SenderID: "ou_self", Text: "我先整理方案"},
	}

	contexts := c.collectChatContexts(context.Background(), anchors, "2026-03-10T12:00:00+08:00", "2026-04-10T12:00:00+08:00", "ou_self")
	if len(contexts) != 2 {
		t.Fatalf("contexts = %#v", contexts)
	}
	for _, ctx := range contexts {
		if len(ctx.ContextMessages) < 2 {
			t.Fatalf("context should include fetched history rather than anchors only: %#v", ctx)
		}
	}
}

type concurrentExtraRunner struct {
	mu           sync.Mutex
	calls        [][]string
	started      int
	release      chan struct{}
	errorMessage string
}

func newConcurrentExtraRunner(errorMessage string) *concurrentExtraRunner {
	return &concurrentExtraRunner{
		release:      make(chan struct{}),
		errorMessage: errorMessage,
	}
}

func (r *concurrentExtraRunner) Run(ctx context.Context, args []string) (sandbox.Output, error) {
	r.mu.Lock()
	r.calls = append(r.calls, append([]string(nil), args...))
	r.started++
	started := r.started
	release := r.release
	r.mu.Unlock()
	if started == 2 {
		close(release)
	}
	select {
	case <-release:
	case <-time.After(100 * time.Millisecond):
		return sandbox.Output{}, errors.New("extra fetches did not overlap")
	case <-ctx.Done():
		return sandbox.Output{}, ctx.Err()
	}
	ref := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--doc" || args[i] == "--message-id" {
			ref = args[i+1]
			break
		}
	}
	if ref == "beta" && r.errorMessage != "" {
		return sandbox.Output{}, errors.New(r.errorMessage)
	}
	return sandbox.Output{Stdout: `{"ref":"` + ref + `"}`}, nil
}

func TestRunExtraFetchesRunsConcurrentlyAndPreservesOrder(t *testing.T) {
	runner := newConcurrentExtraRunner("")
	c := New(runner)

	results, err := c.runExtraFetches(context.Background(), "docs_content", []string{"alpha", "beta", "gamma"}, func(ref string) []string {
		return []string{"docs", "+fetch", "--doc", ref, "--format", "json"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %#v", results)
	}
	for i, ref := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(results[i].Command, ref) {
			t.Fatalf("results order should follow refs: %#v", results)
		}
		if results[i].Error != "" {
			t.Fatalf("unexpected fetch error for %s: %#v", ref, results[i])
		}
	}
}

func TestRunExtraFetchesKeepsOtherResultsWhenOneFetchFails(t *testing.T) {
	runner := newConcurrentExtraRunner("fetch failed")
	c := New(runner)

	results, err := c.runExtraFetches(context.Background(), "mail_content", []string{"alpha", "beta", "gamma"}, func(ref string) []string {
		return []string{"mail", "+message", "--message-id", ref, "--format", "json"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %#v", results)
	}
	if results[1].Error != "fetch failed" {
		t.Fatalf("failed fetch should stay attached to its result: %#v", results)
	}
	if results[0].Error != "" || results[2].Error != "" {
		t.Fatalf("successful fetches should still succeed: %#v", results)
	}
}

func TestBuildAnalysisPromptUsesCleanedRawDomains(t *testing.T) {
	bundle := Bundle{Domains: []DomainResult{
		{Domain: "chat", Stdout: `{"contexts":[{"chat_id":"oc_chat_1","messages":[{"text":"我来同步接口进度"}]}]}`, Error: strings.Repeat("z", 600)},
		{Domain: "docs", Stdout: strings.Repeat("b", 2000)},
		{Domain: "docs_content", Stdout: strings.Repeat("g", 2000)},
		{Domain: "calendar", Stdout: "calendar raw text"},
		{Domain: "task", Stdout: strings.Repeat("d", 2000)},
		{Domain: "mail", Stdout: "mail raw text"},
		{Domain: "vc", Stdout: "vc raw text"},
	},
		ChatSummary: ChatSummary{
			RelationshipSummary: "和核心协作者保持高密度 direct message 互动，同时在跨团队群承担信息收敛。",
			CoreCollaborators: []SummaryItem{
				{DisplayName: "小李", Identifier: "ou_core_1", Summary: "方案推进搭档", Evidence: "连续两周 direct message 高频往返。"},
			},
			FrequentPeople: []SummaryItem{
				{DisplayName: "小李", Identifier: "ou_core_1", Summary: "高频技术讨论对象", Evidence: "问题拆解和排期沟通都集中出现。"},
			},
			FrequentChats: []SummaryItem{
				{DisplayName: "跨团队项目群", Identifier: "oc_chat_1", Summary: "高频同步群", Evidence: "多次出现关键结论同步。"},
			},
		},
		DocsSummary: DocsSummary{
			SelectionRule: "先取当前用户创建的文档 10 篇，再补最近浏览 20 篇，合并去重。",
			PersonalDocs: []DocSummary{
				{Title: "个人周报", Identifier: "doc_beta", Reason: "created_by_self"},
			},
			RecentDocs: []DocSummary{
				{Title: "最近浏览设计文档", Identifier: "doc_gamma", Reason: "recent_open"},
			},
			SelectedDocs: []DocSummary{
				{Title: "个人周报", Identifier: "doc_beta", Reason: "merged"},
				{Title: "最近浏览设计文档", Identifier: "doc_gamma", Reason: "merged"},
			},
		},
		AnalysisInput: AnalysisInput{
			Stats: map[string]DomainDigestStats{
				"chat_prompt": {ItemsBefore: 12, ItemsAfter: 7, FilteredItems: 5, FetchedItems: 2},
			},
		},
	}
	bundle.Domains = append(bundle.Domains,
		DomainResult{Domain: "mail_content", Stdout: "mail content raw text"},
		DomainResult{Domain: "vc_notes", Stdout: "vc notes raw text"},
	)

	prompt := BuildAnalysisPrompt(bundle, persona.All())
	for _, want := range []string{
		"不要展示思考过程",
		"只输出合法 JSON",
		"primary_persona",
		"阶段 1：行为事实提取",
		"优先从 docs_content 提取稳定的工作流、文档写作风格、知识积累、显式观点和决策线索",
		"当 chat 与 docs 同时存在时，优先用 docs 作为长期稳定信号，chat 作为即时互动信号",
		"在 work_profile.doc_writing_style、knowledge_signals、work_preferences 等字段里优先引用文档证据",
		"阶段 2：跨域交叉分析",
		"阶段 3：人格匹配与个性化输出",
		"反事实校验",
		"work_profile",
		"expression_fingerprint",
		"output_style",
		"knowledge_signals",
		"evidence 必须是对象数组",
		"信息处理",
		"风险态度",
		"frequent_chats 只能描述工作群、项目群、专题群等群聊信号",
		"绝对不要写入 frequent_chats",
		"## BSPI Catalog",
		"PRISM",
		"变色龙",
		"summary 必须包含最显著的区分性行为",
		"evidence 至少提供 4 条",
		"优先引用跨域一致信号",
		"如果证据不足",
		"不要编造没有出现在授权数据中的事实",
		"避免使用带评判色彩或过度拟人化的措辞",
		"confidence 必须是 0 到 1 之间的数字",
		"interaction_insights",
		"relationship_summary",
		"SelectionRule",
		"至少 2 个“该用户独有或高度区分”的具体行为",
		"至少 2 条必须引用原话或原文片段",
		"若证据主要来自 docs",
		"chat 信号稀薄或偏噪声",
		"协作方式：chat 的单聊/群聊推进方式，docs 的协作文档痕迹，calendar 的会议角色",
		"## chat",
		"chat_stats: 最近 30 天原始因果链上下文消息 12 条，保留 7 条，过滤 5 条，涉及 2 个会话。",
		"## docs",
		"## docs_content",
		"## task",
		"## Authorized Data",
		"## calendar",
		"## mail",
		"## mail_content",
		"## vc",
		"## vc_notes",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing instruction %q", want)
		}
	}
	for _, digest := range []string{"chat_digest", "docs_digest", "task_digest", "calendar_digest", "mail_digest", "vc_digest"} {
		if strings.Contains(prompt, "## "+digest) {
			t.Fatalf("prompt should not include digest %s", digest)
		}
	}
	for _, raw := range []string{`"我来同步接口进度"`, strings.Repeat("b", 2000), strings.Repeat("g", 2000), strings.Repeat("d", 2000)} {
		if !strings.Contains(prompt, raw) {
			t.Fatalf("prompt should keep raw domain content %q", raw)
		}
	}
	if !strings.Contains(prompt, "calendar raw text") || !strings.Contains(prompt, "vc raw text") || !strings.Contains(prompt, "mail content raw text") {
		t.Fatal("prompt should include raw calendar/vc/mail content")
	}
	if strings.Contains(prompt, "error: "+strings.Repeat("z", 600)) {
		t.Fatal("prompt should not keep raw domain error blocks in the final prompt")
	}
}

func TestBuildAnalysisPromptSanitizesStructuredDomains(t *testing.T) {
	bundle := Bundle{
		Domains: []DomainResult{
			{Domain: "calendar", Stdout: `{"ok":true,"identity":"user","data":[{"summary":"知商周会","description":"","start_time":{"datetime":"2026-03-23T14:00:00+08:00","timezone":"Asia/Shanghai"},"end_time":{"datetime":"2026-03-23T15:00:00+08:00","timezone":"Asia/Shanghai"},"event_id":"evt_1","event_organizer":{"display_name":"闫家宾","user_id":"ou_x"},"app_link":"https://example.com","recurring_event_id":"rec_1","free_busy_status":"busy","self_rsvp_status":"accept"}],"_notice":{"update":{"message":"new version"}}}`},
			{Domain: "mail", Stdout: `[{"date":"2026-04-20T11:05:34Z","from":"ByteDance SSO <bdsso-noreply@mail.bytedance.net>","labels":"","message_id":"mid-1","subject":"你的SSO验证码: 248853","thread_id":"tid-1"}]`},
			{Domain: "docs_content", Stdout: `{"ok":true,"identity":"user","data":{"doc_id":"doc_1","log_id":"log_1","length":12,"offset":0,"total_length":12,"title":"个人周报","markdown":"# 个人周报\n- 推进结论","message":"Document fetched successfully"},"_notice":{"update":{"message":"new version"}}}`},
			{Domain: "vc", Stdout: `{"ok":true,"identity":"user","data":{"has_more":true,"items":[{"display_info":"04-20 | 数据工程周例会","id":"7630794557034187729","meta_data":{"app_link":"https://example.com","avatar":"https://example.com/a.png","description":"昨天 19:01 | 组织者：王雨生"}}]},"_notice":{"update":{"message":"new version"}}}`},
			{Domain: "mail_content", Stdout: `{"body":"建议把风险和回滚方案写进文档。","_notice":{"update":{"message":"new version"}}}`},
			{Domain: "vc_notes", Stdout: `{"notes":"结论：先补压测","_notice":{"update":{"message":"new version"}}}`},
		},
	}

	prompt := BuildAnalysisPrompt(bundle, nil)
	for _, unwanted := range []string{`"event_id"`, `"user_id"`, `"app_link"`, `"recurring_event_id"`, `"message_id"`, `"thread_id"`, `"doc_id"`, `"log_id"`, `"length"`, `"offset"`, `"total_length"`, `"avatar"`, `"id": "7630794557034187729"`, `"update"`} {
		if strings.Contains(prompt, unwanted) {
			t.Fatalf("prompt should sanitize %q: %s", unwanted, prompt)
		}
	}
	for _, want := range []string{`"summary": "知商周会"`, `"display_name": "闫家宾"`, `"free_busy_status": "busy"`, `"self_rsvp_status": "accept"`, `"subject": "你的SSO验证码: 248853"`, `"title": "个人周报"`, `"markdown": "# 个人周报\n- 推进结论"`, `"display_info": "04-20 | 数据工程周例会"`, `"description": "昨天 19:01 | 组织者：王雨生"`, `"body": "建议把风险和回滚方案写进文档。"`, `"notes": "结论：先补压测"`} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing sanitized field %q: %s", want, prompt)
		}
	}
}

func TestBuildAnalysisInputKeepsStatsWithoutDigests(t *testing.T) {
	bundle := Bundle{Domains: []DomainResult{
		{Domain: "chat", Stdout: `{"contexts":[{"chat_id":"oc_chat_1","messages":[{"text":"我来同步接口进度"}]}]}`},
		{Domain: "docs_content", Stdout: `{"data":{"title":"个人周报","markdown":"# 个人周报\n<image token=\"x\"/>\n- 推进结论\n- 风险边界\n"}}`},
		{Domain: "mail", Stdout: `{"items":[{"message_id":"mid-1","subject":"方案评审反馈","from":"teammate@example.com"},{"message_id":"mid-2","subject":"GitHub notification","from":"noreply@github.com"}]}`},
		{Domain: "mail_content", Command: "mail +message --message-id mid-1 --format json", Stdout: `{"body":"建议把风险和回滚方案写进文档，再给出明确上线窗口。"}`},
		{Domain: "mail_content", Command: "mail +message --message-id mid-2 --format json", Stdout: `{"body":"notification digest"}`},
		{Domain: "calendar", Stdout: `{"items":[{"title":"技术评审会","organizer":"Self User"}]}`},
		{Domain: "vc", Stdout: `{"items":[{"title":"周会同步"}]}`},
		{Domain: "vc_notes", Stdout: `{"notes":"结论：先补压测；行动项：周三前完成"}`},
	}}

	filteredMail, mailStats := filterMailDomain(bundle.Domains[2])
	bundle.Domains[2] = filteredMail
	bundle.Domains = []DomainResult{
		bundle.Domains[0],
		bundle.Domains[1],
		bundle.Domains[2],
		bundle.Domains[3],
		bundle.Domains[5],
		bundle.Domains[6],
		bundle.Domains[7],
	}
	input := BuildAnalysisInput(bundle, DomainDigestStats{RawChars: 20, ItemsBefore: 12, ItemsAfter: 7, FilteredItems: 5, FetchedItems: 2}, mailStats)
	if _, ok := input.Stats["chat_digest"]; ok {
		t.Fatalf("chat digest stats should be absent: %#v", input.Stats)
	}
	if _, ok := input.Stats["docs_digest"]; ok {
		t.Fatalf("docs digest stats should be absent: %#v", input.Stats)
	}
	if _, ok := input.Stats["task_digest"]; ok {
		t.Fatalf("task digest stats should be absent: %#v", input.Stats)
	}
	if input.Stats["chat_prompt"].ItemsAfter != 7 {
		t.Fatalf("chat prompt stats = %#v", input.Stats["chat_prompt"])
	}
	if input.Stats["mail_filter"].ItemsBefore != 2 || input.Stats["mail_filter"].ItemsAfter != 1 || input.Stats["mail_filter"].FetchedItems != 1 {
		t.Fatalf("mail filter stats = %#v", input.Stats["mail_filter"])
	}
	if input.Stats["docs_content_raw"].FetchedItems != 1 {
		t.Fatalf("docs content stats = %#v", input.Stats["docs_content_raw"])
	}
}

func TestBuildAnalysisInputUsesOriginalChatRawChars(t *testing.T) {
	bundle := Bundle{Domains: []DomainResult{
		{Domain: "chat", Stdout: ""},
	}}

	input := BuildAnalysisInput(bundle, DomainDigestStats{RawChars: 128, FetchedItems: 2}, DomainDigestStats{})
	if input.Stats["chat_raw"].RawChars != 128 {
		t.Fatalf("chat raw chars = %#v", input.Stats["chat_raw"])
	}
	if input.Stats["chat_raw"].FetchedItems != 1 {
		t.Fatalf("chat raw fetched items = %#v", input.Stats["chat_raw"])
	}
}

func TestFilterMailDomainRemovesSystemNotifications(t *testing.T) {
	result := DomainResult{
		Domain: "mail",
		Stdout: `{"items":[{"message_id":"mid-1","subject":"方案评审反馈","from":"teammate@example.com"},{"message_id":"mid-2","subject":"GitHub notification","from":"noreply@github.com"}]}`,
	}
	filtered, stats := filterMailDomain(result)
	if strings.Contains(strings.ToLower(filtered.Stdout), "github") || strings.Contains(strings.ToLower(filtered.Stdout), "noreply") {
		t.Fatalf("filtered mail should exclude system notifications: %q", filtered.Stdout)
	}
	if !strings.Contains(filtered.Stdout, "方案评审反馈") {
		t.Fatalf("filtered mail should keep human mail: %q", filtered.Stdout)
	}
	if stats.ItemsBefore != 2 || stats.ItemsAfter != 1 || stats.FilteredItems != 1 {
		t.Fatalf("mail filter stats = %#v", stats)
	}
}

func TestFilterMailByBodiesRemovesVerificationMail(t *testing.T) {
	mailResult := DomainResult{
		Domain: "mail",
		Stdout: `{"items":[
			{"message_id":"mid-1","subject":"登录提醒","from":"service@example.com"},
			{"message_id":"mid-2","subject":"方案评审反馈","from":"teammate@example.com"}
		]}`,
	}
	mailContent := []DomainResult{
		{Domain: "mail_content", Command: "mail +message --message-id mid-1 --format json", Stdout: `{"body":"你的验证码是 123456，请在 5 分钟内完成验证。"}`},
		{Domain: "mail_content", Command: "mail +message --message-id mid-2 --format json", Stdout: `{"body":"建议把风险和回滚方案写进文档，再给出明确上线窗口。"}`},
	}

	filteredMail, filteredContent := filterMailByBodies(mailResult, mailContent)
	if strings.Contains(filteredMail.Stdout, "mid-1") || strings.Contains(filteredMail.Stdout, "验证码") {
		t.Fatalf("verification mail should be removed from mail domain: %q", filteredMail.Stdout)
	}
	if len(filteredContent) != 1 {
		t.Fatalf("filtered mail content = %#v", filteredContent)
	}
	if strings.Contains(filteredContent[0].Stdout, "验证码") {
		t.Fatalf("verification mail should be removed from mail content: %#v", filteredContent)
	}
	if !strings.Contains(filteredContent[0].Command, "mid-2") {
		t.Fatalf("remaining mail content should be human mail: %#v", filteredContent)
	}
}

func TestIsHighSignalChatMessage(t *testing.T) {
	for _, tc := range []struct {
		text string
		want bool
	}{
		{text: "好的", want: false},
		{text: "收到", want: false},
		{text: "我来同步接口进度", want: true},
		{text: "那个bug你看了吗", want: true},
		{text: "周三 10 点评审", want: true},
		{text: "https://example.com/spec", want: true},
		{text: "@小李 看下接口", want: true},
	} {
		if got := isHighSignalChatMessage(tc.text); got != tc.want {
			t.Fatalf("isHighSignalChatMessage(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestBuildChatPromptJSONFiltersLowValueMessagesAndKeepsAnchors(t *testing.T) {
	contexts := []chatContext{
		{
			ChatID:      "oc_chat_1",
			ChatName:    "项目群",
			ChatType:    "group",
			Members:     []string{"Self User", "小王", "小李"},
			MemberCount: 3,
			Anchors: []messageCandidate{
				{MessageID: "m2", SenderID: "ou_self", Text: "我来同步接口进度", CreatedAt: 2},
			},
			ContextMessages: []messageCandidate{
				{MessageID: "m1", SenderID: "ou_peer", SenderName: "小王", Text: "收到", CreatedAt: 1},
				{MessageID: "m2", SenderID: "ou_self", SenderName: "Self User", Text: "[回应他人] 我来同步接口进度", CreatedAt: 2, UserAuthored: true},
				{MessageID: "m3", SenderID: "ou_peer", SenderName: "小王", Text: "周三 10 点评审", CreatedAt: 3},
				{MessageID: "m4", SenderID: "ou_peer", SenderName: "小王", Text: "https://example.com/spec", CreatedAt: 4},
			},
		},
		{
			ChatName:        "与小李单聊",
			ChatType:        "p2p",
			CounterpartName: "刘岳林",
			ContextMessages: []messageCandidate{
				{MessageID: "p1", SenderName: "Self User", Text: "我先整理方案", CreatedAt: 10},
			},
		},
	}

	raw, before, after := buildChatPromptJSON(contexts)
	if before != 5 || after != 3 {
		t.Fatalf("counts = before %d after %d", before, after)
	}
	if strings.Contains(raw, `"收到"`) {
		t.Fatalf("low-value confirmation should be removed: %s", raw)
	}
	if strings.Contains(raw, `"https://example.com/spec"`) {
		t.Fatalf("standalone url should be removed: %s", raw)
	}
	for _, want := range []string{`"[回应他人] 我来同步接口进度"`, `"周三 10 点评审"`, `"members": [`, `"member_count": 3`, `"counterpart_name": "刘岳林"`} {
		if !strings.Contains(raw, want) {
			t.Fatalf("chat prompt payload missing %q: %s", want, raw)
		}
	}
	for _, unwanted := range []string{`"anchor_message_ids"`, `"message_id"`, `"sender_id"`, `"chat_id"`, `"created_at"`, `"user_authored"`} {
		if strings.Contains(raw, unwanted) {
			t.Fatalf("chat prompt payload should not contain %q: %s", unwanted, raw)
		}
	}
}

func TestBuildChatPromptJSONTruncatesLargeGroupMembersAndDropsLowValueAnchors(t *testing.T) {
	members := make([]string, 0, 25)
	for i := 1; i <= 25; i++ {
		members = append(members, "成员"+strconv.Itoa(i))
	}

	contexts := []chatContext{
		{
			ChatName:    "超大项目群",
			ChatType:    "group",
			Anchors:     []messageCandidate{{MessageID: "m_anchor", SenderID: "ou_self", Text: "走吗", CreatedAt: 2}},
			Members:     members,
			MemberCount: 25,
			ContextMessages: []messageCandidate{
				{MessageID: "m_anchor", SenderName: "Self User", Text: "[群发起话题] 走吗", CreatedAt: 2},
				{MessageID: "m_keep", SenderName: "小王", Text: "周三 10 点评审", CreatedAt: 3},
			},
		},
	}

	raw, before, after := buildChatPromptJSON(contexts)
	if before != 2 || after != 1 {
		t.Fatalf("counts = before %d after %d", before, after)
	}
	if strings.Contains(raw, `"走吗"`) {
		t.Fatalf("low-value anchor should be removed from chat prompt: %s", raw)
	}
	if !strings.Contains(raw, `"member_count": 25`) {
		t.Fatalf("chat prompt payload missing member_count: %s", raw)
	}
	if !strings.Contains(raw, `"成员10"`) || strings.Contains(raw, `"成员11"`) {
		t.Fatalf("group members should be truncated to top 10: %s", raw)
	}
}

func TestExtractMessageCandidatesParsesNestedSenderFields(t *testing.T) {
	raw := `{
		"ok": true,
		"data": {
			"messages": [
				{
					"chat_id": "oc_chat_1",
					"chat_type": "p2p",
					"content": "我来同步接口进度",
					"create_time": "1710000000",
					"message_id": "om_msg_1",
					"sender": {
						"id": "ou_self",
						"name": "覃晔"
					}
				}
			]
		}
	}`

	messages := extractMessageCandidates(raw)
	if len(messages) == 0 {
		t.Fatal("expected parsed messages")
	}
	if messages[0].SenderID != "ou_self" {
		t.Fatalf("sender id = %q", messages[0].SenderID)
	}
	if messages[0].SenderName != "覃晔" {
		t.Fatalf("sender name = %q", messages[0].SenderName)
	}
	if messages[0].ChatID != "oc_chat_1" || messages[0].Text != "我来同步接口进度" {
		t.Fatalf("parsed message = %#v", messages[0])
	}
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func hasArgValue(args []string, key, want string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == want {
			return true
		}
	}
	return false
}

func findCall(calls [][]string, prefix ...string) ([]string, bool) {
	for _, call := range calls {
		if len(call) < len(prefix) {
			continue
		}
		matched := true
		for i, want := range prefix {
			if call[i] != want {
				matched = false
				break
			}
		}
		if matched {
			return call, true
		}
	}
	return nil, false
}

func cloneCalls(calls [][]string) [][]string {
	cloned := make([][]string, 0, len(calls))
	for _, call := range calls {
		cloned = append(cloned, append([]string(nil), call...))
	}
	return cloned
}

func writeSessionUserConfig(t *testing.T, dir, openID, name string) {
	t.Helper()
	cliDir := filepath.Join(dir, "lark-cli")
	if err := os.MkdirAll(cliDir, 0700); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"apps": []map[string]any{
			{
				"appId": "cli_fake",
				"appSecret": map[string]string{
					"source": "file",
					"id":     filepath.Join(cliDir, "app_secret"),
				},
				"brand": "feishu",
				"lang":  "zh",
				"users": []map[string]string{
					{
						"userOpenId": openID,
						"userName":   name,
					},
				},
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "config.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
