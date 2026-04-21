package collector

type SummaryItem struct {
	DisplayName string `json:"display_name"`
	Identifier  string `json:"identifier,omitempty"`
	Summary     string `json:"summary"`
	Evidence    string `json:"evidence"`
}

type ChatSummary struct {
	RelationshipSummary string        `json:"relationship_summary"`
	CoreCollaborators   []SummaryItem `json:"core_collaborators"`
	FrequentPeople      []SummaryItem `json:"frequent_people"`
	FrequentChats       []SummaryItem `json:"frequent_chats"`
}

type DomainDigestStats struct {
	RawChars      int `json:"raw_chars"`
	DigestChars   int `json:"digest_chars"`
	ItemsBefore   int `json:"items_before,omitempty"`
	ItemsAfter    int `json:"items_after,omitempty"`
	FilteredItems int `json:"filtered_items,omitempty"`
	FetchedItems  int `json:"fetched_items,omitempty"`
}

type AnalysisInput struct {
	Stats map[string]DomainDigestStats `json:"stats,omitempty"`
}

type DocSummary struct {
	Title      string `json:"title"`
	Identifier string `json:"identifier,omitempty"`
	URL        string `json:"url,omitempty"`
	Reason     string `json:"reason"`
}

type DocsSummary struct {
	SelectionRule     string       `json:"selection_rule"`
	ApproximationNote string       `json:"approximation_note,omitempty"`
	PersonalDocs      []DocSummary `json:"personal_docs"`
	RecentDocs        []DocSummary `json:"recent_docs"`
	SelectedDocs      []DocSummary `json:"selected_docs"`
}
