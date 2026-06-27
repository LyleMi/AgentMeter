const OK_STATUSES = new Set(['completed', 'ok', 'indexed', 'success'])
const WARNING_STATUSES = new Set(['pending', 'warning', 'scanning', 'unknown', 'started'])

export function normalizedStatus(status?: string) {
  return (status || 'unknown').toLowerCase()
}

export function statusClass(status?: string) {
  const normalized = normalizedStatus(status)
  if (OK_STATUSES.has(normalized)) return 'status-ok'
  if (WARNING_STATUSES.has(normalized)) return 'status-warning'
  return 'status-error'
}

export function statusColor(status?: string) {
  const normalized = normalizedStatus(status)
  if (OK_STATUSES.has(normalized)) return 'success'
  if (normalized === 'scanning') return 'processing'
  if (WARNING_STATUSES.has(normalized)) return 'warning'
  return 'error'
}
