import { useState } from 'react'
import { patch, post } from '../../api'
import { Alert, Card, Chip, CopyButton, Empty, Field, PageHeader, Tip } from '../../components/ui.jsx'
import { titleCase } from '../../format'

const UUID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/

export default function AdminUsers() {
  const [form, setForm] = useState({ username: '', role: 'CUSTOMER', temporaryPassword: '', fullName: '' })
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState(null)
  const [created, setCreated] = useState([])
  const [manage, setManage] = useState({ userId: '', status: 'ACTIVE', role: 'CUSTOMER', newPassword: '' })
  const [manageMsg, setManageMsg] = useState('')
  const [manageError, setManageError] = useState(null)
  const [mBusy, setMBusy] = useState('')

  const fe = (k) => createError && createError.fields && createError.fields[k]
  const customer = form.role === 'CUSTOMER'

  async function create(e) {
    e.preventDefault()
    setCreating(true)
    setCreateError(null)
    try {
      const body = { username: form.username.trim(), role: form.role, temporaryPassword: form.temporaryPassword }
      if (customer) body.fullName = form.fullName.trim()
      const res = await post('/admin/users', body)
      setCreated([res.data, ...created])
      setForm({ ...form, username: '', temporaryPassword: '', fullName: '' })
    } catch (err) {
      setCreateError(err)
    }
    setCreating(false)
  }

  async function act(kind, fn, okText) {
    setMBusy(kind)
    setManageError(null)
    setManageMsg('')
    try {
      await fn()
      setManageMsg(okText)
      if (kind !== 'password') setCreated((list) => list.map((u) => (u.id === manage.userId.trim() ? { ...u, ...(kind === 'status' ? { status: manage.status } : { role: manage.role }) } : u)))
      if (kind === 'password') setManage((m) => ({ ...m, newPassword: '' }))
    } catch (err) {
      setManageError(err)
    }
    setMBusy('')
  }

  const id = manage.userId.trim()
  const idOk = UUID_RE.test(id)
  return (
    <>
      <PageHeader title="Users" subtitle="Create staff and customer logins, and manage existing ones." />
      <div className="two-col">
        <div>
          <Card title="Create a user">
            <form className="stack" onSubmit={create} noValidate>
              <Alert error={createError && !createError.fields ? createError : null} />
              <Field label="Username" htmlFor="u-name" error={fe('username')}><input id="u-name" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} autoComplete="off" /></Field>
              <Field label="Role" htmlFor="u-role" error={fe('role')} hint="Administrators cannot be created here.">
                <select id="u-role" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
                  <option value="CUSTOMER">Customer</option><option value="OPERATIONS">Operations</option>
                </select>
              </Field>
              {customer && <Field label="Full name" htmlFor="u-full" error={fe('fullName')}><input id="u-full" value={form.fullName} onChange={(e) => setForm({ ...form, fullName: e.target.value })} /></Field>}
              <Field label="Temporary password" htmlFor="u-pw" error={fe('temporaryPassword')} hint="At least 12 characters. Share it with the user directly, not in this app.">
                <input id="u-pw" type="password" autoComplete="new-password" value={form.temporaryPassword} onChange={(e) => setForm({ ...form, temporaryPassword: e.target.value })} />
              </Field>
              <button className="btn btn-primary" disabled={creating || !form.username || !form.temporaryPassword || (customer && !form.fullName)}>{creating ? 'Creating…' : 'Create user'}</button>
            </form>
          </Card>
          <Card title="Created in this session" flush>
            {created.length === 0 ? (
              <Empty title="Nothing yet">Users you create appear here so you can copy their ID.</Empty>
            ) : (
              <div className="table-wrap">
                <table className="table">
                  <thead><tr><th>Username</th><th>Role</th><th>Status</th><th>User ID</th></tr></thead>
                  <tbody>
                    {created.map((u) => (
                      <tr key={u.id}>
                        <td className="strong">{u.username}</td><td>{titleCase(u.role)}</td><td><Chip value={u.status} /></td>
                        <td><code>{u.id}</code> <CopyButton text={u.id} /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </div>
        <div>
          <Card title="Manage a user">
            <div className="stack">
              <Alert kind="ok">{manageMsg}</Alert>
              <Alert error={manageError} />
              <Field label="User ID" htmlFor="m-id" error={id && !idOk ? 'A user ID is 36 characters with dashes.' : ''}><input id="m-id" value={manage.userId} onChange={(e) => setManage({ ...manage, userId: e.target.value })} spellCheck="false" /></Field>
              <Field label="Status" htmlFor="m-st">
                <select id="m-st" value={manage.status} onChange={(e) => setManage({ ...manage, status: e.target.value })}><option value="ACTIVE">Active</option><option value="DEACTIVATED">Deactivated</option></select>
              </Field>
              <button className="btn btn-quiet" disabled={!idOk || mBusy} onClick={() => act('status', () => patch(`/admin/users/${id}/status`, { status: manage.status }), 'Status updated.')}>Update status</button>
              <Field label="Role" htmlFor="m-role">
                <select id="m-role" value={manage.role} onChange={(e) => setManage({ ...manage, role: e.target.value })}><option value="CUSTOMER">Customer</option><option value="OPERATIONS">Operations</option></select>
              </Field>
              <button className="btn btn-quiet" disabled={!idOk || mBusy} onClick={() => act('role', () => patch(`/admin/users/${id}/role`, { role: manage.role }), 'Role updated.')}>Change role</button>
              <Field label="New password" htmlFor="m-pw" hint="At least 12 characters.">
                <input id="m-pw" type="password" autoComplete="new-password" value={manage.newPassword} onChange={(e) => setManage({ ...manage, newPassword: e.target.value })} />
              </Field>
              <button className="btn btn-quiet" disabled={!idOk || !manage.newPassword || mBusy} onClick={() => act('password', () => post(`/admin/users/${id}/reset-password`, { newPassword: manage.newPassword }), 'Password reset.')}>Reset password</button>
            </div>
          </Card>
          <Tip title="Why paste an ID?">The bank has no user list endpoint, so this screen can only act on IDs you already have. IDs of users created here are shown on the left.</Tip>
        </div>
      </div>
    </>
  )
}
