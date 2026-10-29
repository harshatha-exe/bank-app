import { useEffect, useRef, useState } from 'react'
import { friendly } from '../api'
import { titleCase } from '../format'

export function Logo({ light }) {
  return (
    <span className={'logo' + (light ? ' logo-light' : '')}>
      <svg width="30" height="30" viewBox="0 0 32 32" aria-hidden="true">
        <rect width="32" height="32" rx="7" fill="#0B1F3A" />
        <path d="M8 22V10h7a4 4 0 0 1 0 8H8m7 0a4 4 0 0 1 0 4H8" fill="none" stroke="#2DD4BF" strokeWidth="2.4" />
      </svg>
      <span className="logo-text">Bank App</span>
    </span>
  )
}

export function PageHeader({ title, subtitle, actions }) {
  return (
    <div className="page-header">
      <div>
        <h1>{title}</h1>
        {subtitle && <p className="subtitle">{subtitle}</p>}
      </div>
      {actions && <div className="page-actions">{actions}</div>}
    </div>
  )
}

export function Card({ title, aside, children, className = '', flush }) {
  return (
    <section className={'card ' + className}>
      {(title || aside) && (
        <header className="card-head">
          <h2>{title}</h2>
          {aside}
        </header>
      )}
      <div className={flush ? 'card-body flush' : 'card-body'}>{children}</div>
    </section>
  )
}

// A short, plain-language hint. Used across the app for "good to know" details.
export function Tip({ title = 'Good to know', children }) {
  return (
    <aside className="tip">
      <span className="tip-label">{title}</span>
      <div>{children}</div>
    </aside>
  )
}

export function Alert({ kind = 'error', children, error }) {
  if (!children && !error) return null
  return (
    <div className={'alert alert-' + kind} role={kind === 'error' ? 'alert' : 'status'}>
      <div>{error ? friendly(error) : children}</div>
      {error && error.correlationId && <div className="alert-ref">Reference {error.correlationId}</div>}
    </div>
  )
}

const CHIP = {
  ACTIVE: 'ok', COMPLETED: 'ok', RESOLVED: 'ok', CREDIT: 'ok',
  FROZEN: 'bad', FAILED: 'bad', DEACTIVATED: 'muted', DEBIT: 'neutral',
  PENDING: 'warn', UNDER_REVIEW: 'warn', NONE: 'muted', SUCCESS: 'ok', FAILURE: 'bad',
}
export function Chip({ value, label }) {
  return <span className={'chip chip-' + (CHIP[value] || 'neutral')}>{label || titleCase(value)}</span>
}

export function Loading({ label = 'Loading…' }) {
  return (
    <div className="loading" role="status">
      <span className="spinner" aria-hidden="true" />
      {label}
    </div>
  )
}

export function Empty({ title, children }) {
  return (
    <div className="empty">
      <strong>{title}</strong>
      {children && <p>{children}</p>}
    </div>
  )
}

export function Field({ label, hint, error, children, htmlFor }) {
  return (
    <div className={'field' + (error ? ' has-error' : '')}>
      <label htmlFor={htmlFor}>{label}</label>
      {children}
      {error ? <div className="field-error">{error}</div> : hint ? <div className="field-hint">{hint}</div> : null}
    </div>
  )
}

export function CopyButton({ text, label = 'Copy' }) {
  const [done, setDone] = useState(false)
  const t = useRef()
  useEffect(() => () => clearTimeout(t.current), [])
  async function copy() {
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      const ta = document.createElement('textarea')
      ta.value = text
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      ta.remove()
    }
    setDone(true)
    t.current = setTimeout(() => setDone(false), 1500)
  }
  return (
    <button type="button" className="btn-link" onClick={copy}>
      {done ? 'Copied' : label}
    </button>
  )
}

export function IdCell({ id }) {
  return (
    <span className="id-cell">
      <code title={id}>{id.slice(0, 8)}…</code>
      <CopyButton text={id} />
    </span>
  )
}

export function Pager({ meta, onPage }) {
  if (!meta || !meta.totalPages || meta.totalPages <= 1) {
    return meta && meta.totalItems != null ? <div className="pager"><span>{meta.totalItems} {meta.totalItems === 1 ? 'record' : 'records'}</span></div> : null
  }
  return (
    <div className="pager">
      <span>
        Page {meta.page} of {meta.totalPages} · {meta.totalItems} records
      </span>
      <div>
        <button className="btn btn-quiet" disabled={meta.page <= 1} onClick={() => onPage(meta.page - 1)}>Previous</button>
        <button className="btn btn-quiet" disabled={meta.page >= meta.totalPages} onClick={() => onPage(meta.page + 1)}>Next</button>
      </div>
    </div>
  )
}

export function ConfirmDialog({ open, title, children, confirmLabel = 'Confirm', busy, onConfirm, onCancel, error }) {
  const ref = useRef()
  useEffect(() => {
    if (open) ref.current?.focus()
    const onKey = (e) => e.key === 'Escape' && open && !busy && onCancel()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, busy, onCancel])
  if (!open) return null
  return (
    <div className="modal-backdrop" onMouseDown={(e) => e.target === e.currentTarget && !busy && onCancel()}>
      <div className="modal" role="dialog" aria-modal="true" aria-label={title}>
        <h2>{title}</h2>
        <div className="modal-body">{children}</div>
        <Alert error={error} />
        <div className="modal-actions">
          <button className="btn btn-quiet" onClick={onCancel} disabled={busy}>Cancel</button>
          <button className="btn btn-primary" ref={ref} onClick={onConfirm} disabled={busy}>
            {busy ? 'Working…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

// Small helper hook: run an async loader, expose { data, meta, error, loading, reload }.
export function useLoad(loader, deps) {
  const [state, setState] = useState({ loading: true, data: null, meta: null, error: null })
  const [tick, setTick] = useState(0)
  useEffect(() => {
    let live = true
    setState((s) => ({ ...s, loading: true, error: null }))
    loader()
      .then((res) => live && setState({ loading: false, data: res.data, meta: res.meta, error: null }))
      .catch((error) => live && setState({ loading: false, data: null, meta: null, error }))
    return () => {
      live = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick])
  return { ...state, reload: () => setTick((t) => t + 1) }
}
