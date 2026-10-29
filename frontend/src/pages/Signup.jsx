import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { post } from '../api'
import { Alert, Field, Logo, Tip } from '../components/ui.jsx'

export default function Signup() {
  const navigate = useNavigate()
  const [form, setForm] = useState({ fullName: '', username: '', password: '' })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const [done, setDone] = useState(false)
  const set = (k) => (e) => setForm({ ...form, [k]: e.target.value })
  const fe = (k) => error && error.fields && error.fields[k]

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await post('/auth/signup', { username: form.username.trim(), password: form.password, fullName: form.fullName.trim() })
      setDone(true)
    } catch (err) {
      setError(err)
    }
    setBusy(false)
  }

  return (
    <div className="auth-page">
      <section className="auth-brand">
        <Logo light />
        <div>
          <h1>Open your customer profile.</h1>
          <p>Sign up takes a minute. Accounts are linked to your profile by the bank afterwards.</p>
        </div>
      </section>
      <section className="auth-panel">
        {done ? (
          <div className="auth-form">
            <h2>Profile created</h2>
            <p className="subtitle">You can sign in now with <strong>{form.username}</strong>.</p>
            <Tip title="No accounts yet">
              A new profile starts without a bank account. Accounts are added by the bank, so your overview will be empty until then.
            </Tip>
            <button className="btn btn-primary btn-block" onClick={() => navigate('/login')}>Go to sign in</button>
          </div>
        ) : (
          <form className="auth-form" onSubmit={submit} noValidate>
            <h2>Create account</h2>
            <p className="subtitle">Customer sign-up</p>
            <Alert error={error && !error.fields ? error : null} />
            <Field label="Full name" htmlFor="fullName" error={fe('fullName')}>
              <input id="fullName" autoComplete="name" value={form.fullName} onChange={set('fullName')} required />
            </Field>
            <Field label="Username" htmlFor="username" error={fe('username')} hint="At least 3 characters. Letters, numbers, dot, dash or underscore.">
              <input id="username" autoComplete="username" value={form.username} onChange={set('username')} required />
            </Field>
            <Field label="Password" htmlFor="password" error={fe('password')} hint="At least 12 characters.">
              <input id="password" type="password" autoComplete="new-password" value={form.password} onChange={set('password')} required />
            </Field>
            <button className="btn btn-primary btn-block" disabled={busy || !form.fullName || !form.username || form.password.length < 12}>
              {busy ? 'Creating…' : 'Create account'}
            </button>
            <p className="auth-alt">Already registered? <Link to="/login">Sign in</Link></p>
          </form>
        )}
      </section>
    </div>
  )
}
