import { useState } from 'react'
import { post } from '../../api'
import { Alert, Card, Field, PageHeader, Tip } from '../../components/ui.jsx'

const UUID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/

export default function AdminAssignments() {
  const [form, setForm] = useState({ operationsUserId: '', accountId: '' })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const [done, setDone] = useState('')
  const ok = UUID_RE.test(form.operationsUserId.trim()) && UUID_RE.test(form.accountId.trim())

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    setDone('')
    try {
      const res = await post('/admin/account-assignments', { operationsUserId: form.operationsUserId.trim(), accountId: form.accountId.trim() })
      setDone(`Assigned. Assignment ID ${res.data.id}.`)
    } catch (err) {
      setError(err)
    }
    setBusy(false)
  }
  const fe = (k) => error && error.fields && error.fields[k]
  return (
    <>
      <PageHeader title="Account assignments" subtitle="Choose which accounts an operations user can see." />
      <div className="two-col">
        <Card title="Assign an account">
          <form className="stack" onSubmit={submit} noValidate>
            <Alert kind="ok">{done}</Alert>
            <Alert error={error && !error.fields ? error : null} />
            <Field label="Operations user ID" htmlFor="a-u" error={fe('operationsUserId')}><input id="a-u" value={form.operationsUserId} onChange={(e) => setForm({ ...form, operationsUserId: e.target.value })} spellCheck="false" /></Field>
            <Field label="Account ID" htmlFor="a-a" error={fe('accountId')}><input id="a-a" value={form.accountId} onChange={(e) => setForm({ ...form, accountId: e.target.value })} spellCheck="false" /></Field>
            <button className="btn btn-primary" disabled={busy || !ok}>{busy ? 'Assigning…' : 'Assign account'}</button>
          </form>
        </Card>
        <Tip title="Why this matters">Operations staff see only the accounts assigned here. Until you assign one, their account and transfer screens stay empty.</Tip>
      </div>
    </>
  )
}
