export type SessionStatus =
  | 'created'
  | 'config_pending'
  | 'login_pending'
  | 'authenticated'
  | 'collecting'
  | 'analyzing'
  | 'done'
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

export interface Analysis {
  summary: string
  evidence: string[]
  communication_style: string
  work_preferences: string
  blind_spots: string
  confidence: string
  disclaimer: string
}

export interface ReportData {
  primary_persona: PrimaryPersona
  analysis: Analysis
  highlight_tags: string[]
  behavior_vectors: BehaviorVector[]
  share_card: ShareCard
  coverage: Coverage
}

export interface SessionStatusResponse {
  session_id: string
  status: SessionStatus
  verification_url?: string
  error?: string
  report_ready: boolean
  primary_persona?: PrimaryPersona
  progress: ProgressView
  events: SessionEvent[]
  next_action: string
}
