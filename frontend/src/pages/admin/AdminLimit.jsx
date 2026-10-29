import { useEffect, useState } from 'react'
import { get, patch } from '../../api'
import { Alert, Card, Field, Loading, PageHeader, Tip, useLoad } from '../../components/ui.jsx'
import { money, MONEY_RE } from '../../format'

export default function AdminLimit() {
  const { loading, data, error, reload } = useLoad(() => get('/admin/config/transaction-limit'), [])
  const [value, setValue] = useState('')
  const [busy, setBusy] = useState(false)
  const [saveError, setSaveError] = useState(null)
  const [done, setDone] = useState('')
  useEffect(() => { if (data) setValue(data.maxTransferAmount) }, [data])
  const valid = MONEY_RE.test(value) && Number(value) > 0

  async function save(e) {
    e.preventDefault()
    setBusy(true)
    setSaveError(null)
    setDone('')
    try {
      const res = await patch('/admin/config/transaction-limit', { maxTransferAmount: value })
      setDone(`Limit is now ${money(res.data.maxTransferAmount)}.`)
      reload()
    } catch (err) {
      setSaveError(err)
    }
    setBusy(false)
  }
  return (
    <>
      <PageHeader title="Transfer limit" subtitle="The largest amount a single transfer may carry." />
      <Alert error={error} />
      {loading && !data ? <Loading /> : data && (
        <div className="two-col">
          <Card title="Current limit">
            <div className="stat-value">{money(data.maxTransferAmount)}</div>
            <form className="stack" onSubmit={save} noValidate style={{ marginTop: 20 }}>
              <Alert kind="ok">{done}</Alert>
              <Alert error={saveError} />
              <Field label="New limit (INR)" htmlFor="lim" error={value && !valid ? 'Enter an amount above zero with at most two decimals.' : ''}>
                <input id="lim" inputMode="decimal" value={value} onChange={(e) => setValue(e.target.value.trim())} />
              </Field>
              <button className="btn btn-primary" disabled={busy || !valid || value === data.maxTransferAmount}>{busy ? 'Saving…' : 'Update limit'}</button>
            </form>
          </Card>
          <Tip title="Takes effect immediately">New transfers are checked against this value. Transfers already completed are not changed. The update is recorded in the audit log.</Tip>
        </div>
      )}
    </>
  )
}
