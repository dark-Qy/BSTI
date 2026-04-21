export type SessionStatus =
  | 'created'
  | 'config_pending'
  | 'login_pending'
  | 'authenticated'
  | 'collecting'
  | 'analyzing'
  | 'done'
  | 'auth_failed'
  | 'analysis_failed'
  | 'failed'

export interface SessionEvent {
  stage: string
  label: string
  percent: number
  timestamp: string
}

export interface ProgressView {
  stage: string
  label: string
  percent: number
}

export interface PrimaryPersona {
  shorthand: string
  chinese_label: string
  image_url: string
  byte_style_dimension: string
  analysis_dimension: string
  one_liner: string
  canonical_description: string
}

export interface BehaviorVector {
  label: string
  left_pole: string
  right_pole: string
  score: number
  summary: string
}

export interface EvidenceItem {
  domains: string[]
  behavior: string
  strength: string
  is_cross_domain: boolean
  is_distinctive: boolean
}

export interface WorkProfile {
  responsibility_scope: string
  typical_workflow: string
  doc_writing_style: string
  decision_making_pattern: string
  tech_stack_or_domain: string[]
}

export interface ExpressionFingerprint {
  catchphrases: string[]
  jargon: string[]
  sentence_pattern: string
  emoji_habit: string
  formality_spectrum: string
  reply_speed_pattern: string
  conflict_expression: string
}

export interface OutputStyle {
  doc_structure_preference: string
  detail_level: string
  email_reply_pattern: string
  chat_reply_pattern: string
  meeting_behavior: string
}

export interface KnowledgeSignals {
  explicit_opinions: string[]
  learned_lessons: string[]
  repeated_concerns: string[]
  reference_sources: string[]
}

export interface Coverage {
  successful_domains: string[]
  failed_domains: string[]
  summary: string
}

export interface ShareCard {
  title: string
  subtitle: string
  image_url: string
  disclaimer_short: string
}

export interface InteractionTarget {
  display_name: string
  summary: string
  evidence: string
}

export interface InteractionInsights {
  relationship_summary: string
  core_collaborators: InteractionTarget[]
  frequent_people: InteractionTarget[]
  frequent_chats: InteractionTarget[]
}

export interface Analysis {
  summary: string
  evidence: EvidenceItem[]
  communication_style: string
  work_preferences: string
  blind_spots: string
  confidence: number
  disclaimer: string
}

export interface ReportData {
  primary_persona: PrimaryPersona
  analysis: Analysis
  work_profile: WorkProfile
  expression_fingerprint: ExpressionFingerprint
  output_style: OutputStyle
  knowledge_signals: KnowledgeSignals
  highlight_tags: string[]
  behavior_vectors: BehaviorVector[]
  interaction_insights: InteractionInsights
  contrast_signals?: string[]
  share_card: ShareCard
  coverage: Coverage
}

export interface SessionStatusResponse {
  session_id: string
  status: SessionStatus
  app_config_required: boolean
  verification_url?: string
  error?: string
  report_ready: boolean
  primary_persona?: PrimaryPersona
  progress: ProgressView
  events: SessionEvent[]
  next_action: string
}
