import type { ReportData, SessionStatusResponse } from './types'

async function decode<T>(response: Response): Promise<T> {
  const data = (await response.json()) as T & { error?: string }
  if (!response.ok) {
    throw new Error(data.error || `Request failed with status ${response.status}`)
  }
  return data
}

export async function createSession(): Promise<{ session_id: string; status: string }> {
  return decode(await fetch('/api/sessions', { method: 'POST' }))
}

export async function startLogin(
  sessionId: string,
): Promise<{ verification_url?: string; status: string }> {
  return decode(await fetch(`/api/sessions/${sessionId}/login`, { method: 'POST' }))
}

export async function analyzeSession(sessionId: string): Promise<{ status: string }> {
  return decode(await fetch(`/api/sessions/${sessionId}/analyze`, { method: 'POST' }))
}

export async function getSessionStatus(sessionId: string): Promise<SessionStatusResponse> {
  return decode(await fetch(`/api/sessions/${sessionId}/status`))
}

export async function getReportData(
  sessionId: string,
): Promise<{ status: string; error?: string; report: ReportData }> {
  return decode(await fetch(`/api/sessions/${sessionId}/report-data`))
}
