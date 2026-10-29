const inr = new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', minimumFractionDigits: 2 })

// Amounts come from the backend as strings like "100000.00". They are only formatted for display.
export function money(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return value ?? '—'
  return inr.format(n)
}

export function dateTime(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d)) return iso
  return d.toLocaleString('en-IN', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export function dateOnly(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d)) return iso
  return d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })
}

export function shortId(id) {
  return id ? id.slice(0, 8) + '…' : '—'
}

export function titleCase(s) {
  if (!s) return ''
  return s.charAt(0) + s.slice(1).toLowerCase().replace(/_/g, ' ')
}

// Same rule the backend uses: whole rupees plus up to two decimals, no sign, no leading zeros.
export const MONEY_RE = /^(0|[1-9][0-9]{0,14})(\.[0-9]{1,2})?$/

export function newIdempotencyKey() {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return 'ui' + Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}

// Turns a "datetime-local" or date input value into RFC 3339 (what the API expects).
export function toRfc3339(value, endOfDay = false) {
  if (!value) return ''
  const d = new Date(value.length === 10 ? value + (endOfDay ? 'T23:59:59' : 'T00:00:00') : value)
  return isNaN(d) ? '' : d.toISOString().replace('.000Z', 'Z')
}
