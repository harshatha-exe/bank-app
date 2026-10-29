import { useState } from 'react'
import { Link } from 'react-router-dom'
import { get, patch, post } from '../../api'
import { Alert, Card, Chip, ConfirmDialog, CopyButton, Empty, Field, Loading, PageHeader, Tip, useLoad } from '../../components/ui.jsx'

const UUID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/

export default function Beneficiaries() {
  const { loading, data, error, reload } = useLoad(() => get('/beneficiaries'), [])
  const [form, setForm] = useState({ destinationAccountId: '', nickname: '' })
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState(null)
  const [notice, setNotice] = useState('')
  const [toggle, setToggle] = useState(null)
  const [toggling, setToggling] = useState(false)
  const [toggleError, setToggleError] = useState(null)

  const idBad = form.destinationAccountId && !UUID_RE.test(form.destinationAccountId.trim())

  async function add(e) {
    e.preventDefault()
    setSaving(true)
    setFormError(null)
    setNotice('')
    try {
      const res = await post('/beneficiaries', { destinationAccountId: form.destinationAccountId.trim(), nickname: form.nickname.trim() })
      setNotice(`${res.data.nickname} (${res.data.destinationAccountMasked}) was added.`)
      setForm({ destinationAccountId: '', nickname: '' })
      reload()
    } catch (err) {
      setFormError(err)
    }
    setSaving(false)
  }

  async function confirmToggle() {
    setToggling(true)
    setToggleError(null)
    try {
      await patch(`/beneficiaries/${toggle.id}/status`, { status: toggle.status === 'ACTIVE' ? 'DEACTIVATED' : 'ACTIVE' })
      setToggle(null)
      reload()
    } catch (err) {
      setToggleError(err)
    }
    setToggling(false)
  }

  const fe = (k) => formError && formError.fields && formError.fields[k]

  return (
    <>
      <PageHeader title="Beneficiaries" subtitle="People and accounts you send money to." actions={<Link className="btn btn-primary" to="/send">Send money</Link>} />
      <div className="two-col">
        <div>
          <Alert error={error} />
          <Card title="Saved beneficiaries" flush>
            {loading ? (
              <Loading />
            ) : data && data.length === 0 ? (
              <Empty title="No beneficiaries yet">Add one with the form to start sending money.</Empty>
            ) : (
              data && (
                <div className="table-wrap">
                  <table className="table">
                    <thead><tr><th>Nickname</th><th>Account</th><th>Status</th><th></th></tr></thead>
                    <tbody>
                      {data.map((b) => (
                        <tr key={b.id}>
                          <td className="strong">{b.nickname}</td>
                          <td>{b.destinationAccountMasked}</td>
                          <td><Chip value={b.status} /></td>
                          <td className="num">
                            <button className="btn btn-quiet btn-sm" onClick={() => { setToggleError(null); setToggle(b) }}>
                              {b.status === 'ACTIVE' ? 'Deactivate' : 'Activate'}
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )
            )}
          </Card>
        </div>
        <div>
          <Card title="Add a beneficiary">
            <form onSubmit={add} className="stack" noValidate>
              <Alert kind="ok">{notice}</Alert>
              <Alert error={formError && !formError.fields ? formError : null} />
              <Field label="Account ID" htmlFor="destId" error={fe('destinationAccountId') || (idBad ? 'This does not look like an account ID. It is 36 characters with dashes.' : '')} hint="Ask the account holder for the account ID shown on their account page.">
                <input id="destId" value={form.destinationAccountId} onChange={(e) => setForm({ ...form, destinationAccountId: e.target.value })} placeholder="00000000-0000-0000-0000-000000000000" spellCheck="false" required />
              </Field>
              <Field label="Nickname" htmlFor="nick" error={fe('nickname')} hint="A name you will recognise, up to 100 characters.">
                <input id="nick" value={form.nickname} onChange={(e) => setForm({ ...form, nickname: e.target.value })} maxLength={100} required />
              </Field>
              <button className="btn btn-primary" disabled={saving || !form.nickname.trim() || !UUID_RE.test(form.destinationAccountId.trim())}>
                {saving ? 'Adding…' : 'Add beneficiary'}
              </button>
            </form>
          </Card>
          <Tip title="Why an account ID?">
            The bank does not offer a lookup by account number, so you add a beneficiary with the account's ID. You cannot add one of your own accounts, and the same account can only be saved once.
          </Tip>
        </div>
      </div>
      <ConfirmDialog
        open={!!toggle}
        title={toggle && toggle.status === 'ACTIVE' ? 'Deactivate beneficiary?' : 'Activate beneficiary?'}
        confirmLabel={toggle && toggle.status === 'ACTIVE' ? 'Deactivate' : 'Activate'}
        busy={toggling}
        error={toggleError}
        onConfirm={confirmToggle}
        onCancel={() => setToggle(null)}
      >
        {toggle && (toggle.status === 'ACTIVE'
          ? <p><strong>{toggle.nickname}</strong> ({toggle.destinationAccountMasked}) will no longer be available for transfers until you activate it again.</p>
          : <p><strong>{toggle.nickname}</strong> ({toggle.destinationAccountMasked}) will be available for transfers again.</p>)}
      </ConfirmDialog>
    </>
  )
}
