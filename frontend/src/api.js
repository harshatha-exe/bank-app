// Thin wrapper around fetch for the bank backend.
// - Sends the session cookie automatically (same origin through the dev proxy).
// - Keeps the CSRF token in memory only and adds it to every POST/PATCH/DELETE.
// - Turns the backend's error JSON into a normal JavaScript Error with .code, .fields, .status.

let csrfToken = null

export function setCsrf(token) {
  csrfToken = token || null
}

export class ApiError extends Error {
  constructor(status, body, fallback) {
    const err = body && body.error
    super((err && err.message) || fallback || 'Something went wrong.')
    this.status = status
    this.code = (err && err.code) || (status === 0 ? 'NETWORK_ERROR' : 'UNKNOWN')
    this.fields = (err && err.fields) || null
    this.correlationId =
      (err && err.correlationId) || (body && body.meta && body.meta.correlationId) || null
    this.retryAfter = null
  }
}

export async function api(method, path, { body, headers, query } = {}) {
  let url = '/api' + path
  if (query) {
    const qs = new URLSearchParams()
    Object.entries(query).forEach(([k, v]) => {
      if (v !== undefined && v !== null && v !== '') qs.set(k, v)
    })
    const s = qs.toString()
    if (s) url += '?' + s
  }
  const h = { Accept: 'application/json', ...(headers || {}) }
  if (body !== undefined) h['Content-Type'] = 'application/json'
  if (method !== 'GET' && csrfToken) h['X-CSRF-Token'] = csrfToken

  let res
  try {
    res = await fetch(url, {
      method,
      headers: h,
      credentials: 'same-origin',
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError(0, null, 'Cannot reach the backend. Is it running on port 8080?')
  }

  let json = null
  if (res.status !== 204) {
    const text = await res.text()
    if (text) {
      try {
        json = JSON.parse(text)
      } catch {
        json = null
      }
    }
  }
  if (!res.ok) {
    const e = new ApiError(res.status, json, `Request failed (${res.status}).`)
    e.retryAfter = res.headers.get('Retry-After')
    throw e
  }
  if (json && json.meta && json.meta.csrfToken) csrfToken = json.meta.csrfToken
  return json || { data: null, meta: {} }
}

export const get = (p, query) => api('GET', p, { query })
export const post = (p, body, headers) => api('POST', p, { body, headers })
export const patch = (p, body) => api('PATCH', p, { body })

// Friendly text for the error codes the backend can return.
const FRIENDLY = {
  INVALID_CREDENTIALS: 'That username or password is not correct.',
  UNAUTHENTICATED: 'Your session has ended. Please sign in again.',
  FORBIDDEN: 'Your role does not have access to this.',
  CSRF_INVALID: 'Your session token is out of date. Refresh the page and try again.',
  RATE_LIMITED: 'Too many attempts in a short time. Please wait a moment and try again.',
  INSUFFICIENT_FUNDS: 'The source account does not have enough balance for this amount.',
  ACCOUNT_FROZEN: 'One of the accounts is frozen, so the transfer cannot go through.',
  BENEFICIARY_INACTIVE: 'This beneficiary is deactivated. Activate it first.',
  TRANSFER_LIMIT_EXCEEDED: 'This amount is above the bank’s per-transfer limit.',
  IDEMPOTENCY_CONFLICT: 'This request key was already used for a different transfer. Start a new transfer.',
  DUPLICATE_BENEFICIARY: 'This account is already saved as a beneficiary.',
  USERNAME_TAKEN: 'That username is already taken.',
  NOT_FOUND: 'We could not find that. Check the ID and try again.',
  VALIDATION_ERROR: 'Some details need fixing.',
  INVALID_REQUEST: 'The backend could not read that request.',
  SEND_UNCERTAIN: 'We could not confirm whether this transfer went through, because the connection dropped. Check Transfers first: if it is listed, do not send again. If it is not listed, press Send again with the same details (within about 10 minutes, in this tab). The bank then treats it as the same request and will not repeat it.',
  NETWORK_ERROR: 'Cannot reach the backend. Is it running on port 8080?',
}

export function friendly(err) {
  if (!err) return ''
  if (err.code === 'RATE_LIMITED' && err.retryAfter) {
    return `${FRIENDLY.RATE_LIMITED} Try again in about ${err.retryAfter} seconds.`
  }
  return FRIENDLY[err.code] || err.message || 'Something went wrong.'
}
