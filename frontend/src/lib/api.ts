/**
 * API helper.
 * - Uses VITE_API_BASE if provided; otherwise falls back to relative path (works with dev proxy).
 */
const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined) || ''

export type AskResponse = {
  answer: string
  winner_index: number
  runners: number
  scores?: number[]              // judge scores; server should return length == runners
  votes_per_candidate?: number[] // kept for compatibility, but we won't use as fallback
  included_indices?: number[]    // indices that actually produced non-empty answers
  consensus_id: string
}

export async function askSwarm(templateId: string | null, payload: string, signal?: AbortSignal) {
  const body = { template_id: templateId || undefined, instruction: payload }
  const res = await fetch(`${API_BASE}/v1/ask`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `HTTP ${res.status}`)
  }
  return res.json() as Promise<AskResponse>
}
