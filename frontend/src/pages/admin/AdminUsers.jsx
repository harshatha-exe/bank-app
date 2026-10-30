import { useState } from 'react'
import { patch, post } from '../../api'
import { get } from '../../api'
import { Alert, Card, Chip, ConfirmDialog, CopyButton, Empty, Field, IdCell, Loading, PageHeader, Pager, useLoad } from '../../components/ui.jsx'
import { titleCase } from '../../format'

const UUID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/
const EMPTY_FILTERS = { search: '', role: '', status: '' }

export default function AdminUsers() {
  const [form, setForm] = useState({ username: '', role: 'CUSTOMER', temporaryPassword: '', fullName: '' })
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState(null)
  const [created, setCreated] = useState([])
  const [manage, setManage] = useState({ userId: '', status: 'ACTIVE', role: 'CUSTOMER', newPassword: '' })
  const [manageMsg, setManageMsg] = useState('')
  const [manageError, setManageError] = useState(null)
  const [mBusy, setMBusy] = useState('')
  const [draft, setDraft] = useState(EMPTY_FILTERS)
  const [applied, setApplied] = useState(EMPTY_FILTERS)
  const [page, setPage] = useState(1)
  const [confirm, setConfirm] = useState('')
  const { loading, data, meta, error, reload } = useLoad(
    () => get('/admin/users', { search: applied.search, role: applied.role, status: applied.status, page, pageSize: 15 }),
    [page, applied],
  )

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
      reload()
    } catch (err) {
      setCreateError(err)
    }
    setCreating(false)
  }

  async function act(kind) {
    setMBusy(kind)
    setManageError(null)
    setManageMsg('')
    try {
      if (kind === 'status') await patch(`/admin/users/${id}/status`, { status: manage.status })
      if (kind === 'role') await patch(`/admin/users/${id}/role`, { role: manage.role })
      if (kind === 'password') await post(`/admin/users/${id}/reset-password`, { newPassword: manage.newPassword })
      setManageMsg(kind === 'status' ? 'Status updated.' : kind === 'role' ? 'Role updated.' : 'Password reset.')
      if (kind === 'password') setManage((m) => ({ ...m, newPassword: '' }))
      setConfirm('')
      reload()
    } catch (err) {
      setManageError(err)
    }
    setMBusy('')
  }

  const id = manage.userId.trim()
  const idOk = UUID_RE.test(id)
  const protectedUser = manage.role === 'ADMIN' && !!manage.username
  const setFilter = (key) => (e) => setDraft({ ...draft, [key]: e.target.value.trim() })
  const selectUser = (user) => {
    setManage({ userId: user.id, username: user.username, status: user.status, role: user.role, newPassword: '' })
    setManageMsg('')
    setManageError(null)
  }
  const confirmTitle = confirm === 'status' ? `${manage.status === 'DEACTIVATED' ? 'Deactivate' : 'Activate'} ${manage.username || id}?` : confirm === 'role' ? `Change ${manage.username || id}'s role?` : `Reset ${manage.username || id}'s password?`
  return (
    <>
      <PageHeader title="Users" subtitle="Create staff and customer logins, and manage existing ones." />
      <div className="two-col">
        <div>
          <Card title="Create a user">
            <form className="stack" onSubmit={create} noValidate>
              <Alert error={createError && !createError.fields ? createError : null} />
              <Field label="Create username" htmlFor="u-name" error={fe('username')}><input id="u-name" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} autoComplete="off" /></Field>
              <Field label="Create role" htmlFor="u-role" error={fe('role')} hint="Administrators cannot be created here.">
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
          <Card title="Find users" flush>
            <form className="filters" onSubmit={(e) => { e.preventDefault(); setPage(1); setApplied(draft) }}>
              <Field label="Find username" htmlFor="find-search"><input id="find-search" value={draft.search} onChange={setFilter('search')} /></Field>
              <Field label="Find role" htmlFor="find-role">
                <select id="find-role" value={draft.role} onChange={setFilter('role')}><option value="">Any</option><option value="CUSTOMER">Customer</option><option value="OPERATIONS">Operations</option><option value="ADMIN">Admin</option></select>
              </Field>
              <Field label="Find status" htmlFor="find-status">
                <select id="find-status" value={draft.status} onChange={setFilter('status')}><option value="">Any</option><option value="ACTIVE">Active</option><option value="DEACTIVATED">Deactivated</option></select>
              </Field>
              <div className="filter-actions">
                <button className="btn btn-primary">Apply</button>
                <button type="button" className="btn btn-quiet" onClick={() => { setDraft(EMPTY_FILTERS); setApplied(EMPTY_FILTERS); setPage(1) }}>Clear</button>
              </div>
            </form>
            <Alert error={error} />
            {loading ? <Loading /> : data && data.length === 0 ? <Empty title="No users match" /> : data && (
              <div className="table-wrap">
                <table className="table">
                  <thead><tr><th>Username</th><th>Role</th><th>Status</th><th>User ID</th><th></th></tr></thead>
                  <tbody>{data.map((user) => (
                    <tr key={user.id}>
                      <td className="strong">{user.username}</td><td>{titleCase(user.role)}</td><td><Chip value={user.status} /></td>
                      <td><IdCell id={user.id} /></td><td className="num"><button className="btn btn-quiet btn-sm" onClick={() => selectUser(user)}>Select</button></td>
                    </tr>
                  ))}</tbody>
                </table>
              </div>
            )}
            <Pager meta={meta} onPage={setPage} />
          </Card>
          <Card title="Manage a user">
            <div className="stack">
              <Alert kind="ok">{manageMsg}</Alert>
              <Alert error={manageError} />
              {manage.username && <div className="strong">Selected user: {manage.username}</div>}
              <Field label="User ID" htmlFor="m-id" error={id && !idOk ? 'A user ID is 36 characters with dashes.' : ''}><input id="m-id" value={manage.userId} onChange={(e) => setManage({ ...manage, userId: e.target.value })} spellCheck="false" /></Field>
              <Field label="Manage status" htmlFor="m-st">
                {protectedUser ? <div id="m-st">{titleCase(manage.status)} (protected)</div> : <select id="m-st" value={manage.status} onChange={(e) => setManage({ ...manage, status: e.target.value })}><option value="ACTIVE">Active</option><option value="DEACTIVATED">Deactivated</option></select>}
              </Field>
              <button className="btn btn-quiet" disabled={!idOk || !!mBusy || protectedUser} onClick={() => { setManageError(null); setConfirm('status') }}>Update status</button>
              <Field label="Manage role" htmlFor="m-role">
                {protectedUser ? <div id="m-role">{titleCase(manage.role)} (protected)</div> : <select id="m-role" value={manage.role} onChange={(e) => setManage({ ...manage, role: e.target.value })}><option value="CUSTOMER">Customer</option><option value="OPERATIONS">Operations</option></select>}
              </Field>
              <button className="btn btn-quiet" disabled={!idOk || !!mBusy || protectedUser} onClick={() => { setManageError(null); setConfirm('role') }}>Change role</button>
              <Field label="New password" htmlFor="m-pw" hint="At least 12 characters.">
                <input id="m-pw" type="password" autoComplete="new-password" value={manage.newPassword} onChange={(e) => setManage({ ...manage, newPassword: e.target.value })} />
              </Field>
              <button className="btn btn-quiet" disabled={!idOk || !manage.newPassword || !!mBusy || protectedUser} onClick={() => { setManageError(null); setConfirm('password') }}>Reset password</button>
            </div>
          </Card>
        </div>
      </div>
      <ConfirmDialog open={!!confirm} title={confirmTitle} confirmLabel="Confirm" busy={!!mBusy} error={manageError} onConfirm={() => act(confirm)} onCancel={() => setConfirm('')}>
        <p>{confirm === 'status' ? `This will set ${manage.username || id} to ${titleCase(manage.status)}.` : confirm === 'role' ? `This will change ${manage.username || id} to ${titleCase(manage.role)}.` : `This will reset the password for ${manage.username || id}.`}</p>
      </ConfirmDialog>
    </>
  )
}
