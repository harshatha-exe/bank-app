import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { get, post } from '../../api'
import { Alert, Card, ConfirmDialog, Field, Loading, PageHeader, Tip, useLoad } from '../../components/ui.jsx'
import { money, MONEY_RE, newIdempotencyKey, titleCase } from '../../format'

// The pending key is kept in this browser tab (sessionStorage) for 10 minutes, so it survives leaving
// the page or a reload while the outcome of a transfer is unclear. It is cleared on success.
const PENDING = 'pendingTransfer'
function loadPending() {
  try {
    const v = JSON.parse(sessionStorage.getItem(PENDING) || 'null')
    if (v && Date.now() - v.at < 10 * 60 * 1000) return v
  } catch { /* ignore */ }
  return { sig: '', key: '', at: 0 }
}

async function loadForm() {
  const [a, b] = await Promise.all([get('/accounts'), get('/beneficiaries')])
  return { data: { accounts: a.data || [], beneficiaries: b.data || [] }, meta: null }
}

export default function SendMoney() {
  const { loading, data, error } = useLoad(loadForm, [])
  const [form, setForm] = useState({ source: '', destination: '', amount: '', purpose: '' })
  const [review, setReview] = useState(false)
  const [busy, setBusy] = useState(false)
  const [sendError, setSendError] = useState(null)
  const [result, setResult] = useState(null)
  // One Idempotency-Key per distinct request. If the same request is retried (for example after a
  // network drop) the same key goes out again, so the bank cannot process it twice.
  const keyRef = useRef(loadPending())

  const active = useMemo(() => (data ? data.beneficiaries.filter((b) => b.status === 'ACTIVE') : []), [data])
  const [mode, setMode] = useState('')
  useEffect(() => {
    if (data && !mode) setMode(active.length > 0 ? 'beneficiary' : 'accounts')
  }, [data, active, mode])
  const source = data && data.accounts.find((a) => a.id === form.source)
  const destinationAccounts = data ? data.accounts.filter((a) => a.id !== form.source) : []
  const dest = mode === 'accounts'
    ? data && data.accounts.find((a) => a.id === form.destination)
    : data && data.beneficiaries.find((b) => b.id === form.destination)
  const hasDestination = mode === 'accounts'
    ? destinationAccounts.some((a) => a.status === 'ACTIVE')
    : active.length > 0
  const amountOk = MONEY_RE.test(form.amount) && Number(form.amount) > 0
  const enough = !source || !amountOk || Number(source.balance) >= Number(form.amount)
  const ready = source && dest && amountOk && form.purpose.length <= 255

  function payload() {
    const body = {
      sourceAccountId: source.id,
      destinationAccountId: mode === 'accounts' ? dest.id : dest.destinationAccountId,
      amount: form.amount,
      purpose: form.purpose.trim(),
    }
    if (mode === 'beneficiary') body.beneficiaryId = dest.id
    return body
  }

  async function send() {
    const body = payload()
    const sig = JSON.stringify(body)
    if (keyRef.current.sig !== sig) keyRef.current = { sig, key: newIdempotencyKey(), at: Date.now() }
    try { sessionStorage.setItem(PENDING, JSON.stringify(keyRef.current)) } catch { /* ignore */ }
    setBusy(true)
    setSendError(null)
    try {
      const res = await post('/transfers', body, { 'Idempotency-Key': keyRef.current.key })
      setResult(res.data)
      try { sessionStorage.removeItem(PENDING) } catch { /* ignore */ }
      keyRef.current = { sig: '', key: '', at: 0 }
      setReview(false)
    } catch (err) {
      setSendError(err.code === 'NETWORK_ERROR' ? Object.assign(new Error('uncertain'), { code: 'SEND_UNCERTAIN' }) : err)
    }
    setBusy(false)
  }

  if (result) {
    return (
      <>
        <PageHeader title="Transfer sent" />
        <Card>
          <div className="success-block">
            <div className="success-mark" aria-hidden="true">✓</div>
            <h2>{money(result.amount)} sent</h2>
            <p>Reference <strong>{result.referenceNumber}</strong> · Status {titleCase(result.status)}</p>
            <div className="row-actions">
              <Link className="btn btn-primary" to={`/transfers/${result.referenceNumber}`}>View transfer</Link>
              <Link className="btn btn-quiet" to="/">Back to overview</Link>
            </div>
          </div>
        </Card>
      </>
    )
  }

  return (
    <>
      <PageHeader title="Send money" subtitle="Move money between your accounts or to a saved beneficiary." />
      <Alert error={error} />
      {loading ? (
        <Loading />
      ) : data ? (
        <div className="two-col">
          <Card title="Transfer details">
            {data.accounts.length === 0 || !hasDestination ? (
              <Alert kind="warn">
                {data.accounts.length === 0
                  ? 'You have no account to send from yet.'
                  : mode === 'beneficiary'
                    ? <>You need at least one active beneficiary. <Link to="/beneficiaries">Add a beneficiary</Link>.</>
                    : 'You need another active account to send money to.'}
              </Alert>
            ) : null}
            <form className="stack" onSubmit={(e) => { e.preventDefault(); setSendError(null); setReview(true) }} noValidate>
              <div className="field">
                <span className="field-label">Send to</span>
                <div role="group" aria-label="Destination type">
                  <button type="button" className={'btn ' + (mode === 'accounts' ? 'btn-primary' : 'btn-quiet')} aria-pressed={mode === 'accounts'} onClick={() => { setMode('accounts'); setForm({ ...form, destination: '' }) }}>My accounts</button>{' '}
                  <button type="button" className={'btn ' + (mode === 'beneficiary' ? 'btn-primary' : 'btn-quiet')} aria-pressed={mode === 'beneficiary'} onClick={() => { setMode('beneficiary'); setForm({ ...form, destination: '' }) }}>Saved beneficiary</button>
                </div>
              </div>
              <Field label="From account" htmlFor="src">
                <select id="src" value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value, destination: e.target.value === form.destination ? '' : form.destination })}>
                  <option value="">Choose an account</option>
                  {data.accounts.map((a) => (
                    <option key={a.id} value={a.id} disabled={a.status !== 'ACTIVE'}>
                      {titleCase(a.accountType)} {a.accountNumberMasked} · {money(a.balance)}{a.status !== 'ACTIVE' ? ' (frozen)' : ''}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label={mode === 'accounts' ? 'To account' : 'To beneficiary'} htmlFor="dest">
                <select id="dest" value={form.destination} onChange={(e) => setForm({ ...form, destination: e.target.value })}>
                  <option value="">Choose a {mode === 'accounts' ? 'destination account' : 'beneficiary'}</option>
                  {mode === 'accounts' ? destinationAccounts.map((a) => (
                    <option key={a.id} value={a.id} disabled={a.status !== 'ACTIVE'}>
                      {titleCase(a.accountType)} {a.accountNumberMasked} · {money(a.balance)}{a.status !== 'ACTIVE' ? ' (frozen)' : ''}
                    </option>
                  )) : active.map((b) => (
                    <option key={b.id} value={b.id}>{b.nickname} · {b.destinationAccountMasked}</option>
                  ))}
                </select>
              </Field>
              <Field label="Amount (INR)" htmlFor="amt" error={form.amount && !amountOk ? 'Enter an amount above zero with at most two decimals, like 2500 or 2500.50.' : !enough ? 'This is more than the available balance in the chosen account.' : ''} hint="Whole rupees or up to two decimals.">
                <input id="amt" inputMode="decimal" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value.trim() })} placeholder="0.00" />
              </Field>
              <Field label="Purpose (optional)" htmlFor="purpose" hint={`${form.purpose.length}/255 characters`}>
                <input id="purpose" maxLength={255} value={form.purpose} onChange={(e) => setForm({ ...form, purpose: e.target.value })} />
              </Field>
              <button className="btn btn-primary" disabled={!ready || !enough}>Review transfer</button>
            </form>
          </Card>
          <div>
            <Tip title="Before you send">
              You will see a summary to confirm first. Transfers above the bank's per-transfer limit are declined, and money cannot move to or from a frozen account.
            </Tip>
            <Tip title="Safe to retry">
              Each transfer carries a request key. If your connection drops and you press Send again with exactly the same details, the bank recognises the request and does not repeat it. This holds in this browser tab for about 10 minutes. Changing any detail makes it a new transfer.
            </Tip>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={review}
        title="Review and confirm"
        confirmLabel={`Send ${amountOk ? money(form.amount) : ''}`}
        busy={busy}
        error={sendError}
        onConfirm={send}
        onCancel={() => setReview(false)}
      >
        {source && dest && (
          <dl className="summary">
            <dt>From</dt><dd>{titleCase(source.accountType)} {source.accountNumberMasked}</dd>
            <dt>To</dt><dd>{mode === 'accounts' ? `${titleCase(dest.accountType)} ${dest.accountNumberMasked}` : `${dest.nickname} · ${dest.destinationAccountMasked}`}</dd>
            <dt>Amount</dt><dd className="strong">{money(form.amount)}</dd>
            {form.purpose && (<><dt>Purpose</dt><dd>{form.purpose}</dd></>)}
          </dl>
        )}
      </ConfirmDialog>
    </>
  )
}
