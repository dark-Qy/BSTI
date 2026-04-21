const ACTIVE_SESSION_STORAGE_KEY = 'bsti.activeSessionId'

export function loadPersistedSessionId() {
  try {
    return window.localStorage.getItem(ACTIVE_SESSION_STORAGE_KEY)
  } catch {
    return null
  }
}

export function persistSessionId(sessionId: string) {
  try {
    window.localStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, sessionId)
  } catch {
    // Ignore unavailable storage and keep the in-memory session alive.
  }
}

export function clearPersistedSessionId() {
  try {
    window.localStorage.removeItem(ACTIVE_SESSION_STORAGE_KEY)
  } catch {
    // Ignore unavailable storage and let the user continue in-memory.
  }
}
