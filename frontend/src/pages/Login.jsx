import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { homePathFor, useAuth } from '../auth.jsx'
import { Alert, Field, Logo, Tip } from '../components/ui.jsx'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const user = await login(username.trim(), password)
      navigate(homePathFor(user.role), { replace: true })
    } catch (err) {
      setError(err)
      setBusy(false)
    }
  }

  return (
    <div className="auth-page">
      <section className="auth-brand">
        <Logo light />
        <div>
          <h1>Banking that stays clear and in your control.</h1>
          <p>See your balances, move money to saved beneficiaries and keep a record of every step.</p>
        </div>
        <ul className="auth-points">
          <li><strong>Every transfer is traceable.</strong> Each one gets a reference number you can search later.</li>
          <li><strong>Safe retries.</strong> Pressing send twice never moves the money twice.</li>
          <li><strong>Role-based access.</strong> Customers, operations staff and admins each see only their own tools.</li>
        </ul>
      </section>
      <section className="auth-panel">
        <form className="auth-form" onSubmit={submit} noValidate>
          <h2>Sign in</h2>
          <p className="subtitle">Use the username and password for your account.</p>
          <Alert error={error} />
          <Field label="Username" htmlFor="username">
            <input id="username" autoComplete="username" autoFocus value={username} onChange={(e) => setUsername(e.target.value)} required />
          </Field>
          <Field label="Password" htmlFor="password">
            <input id="password" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
          </Field>
          <button className="btn btn-primary btn-block" disabled={busy || !username || !password}>
            {busy ? 'Signing in…' : 'Sign in'}
          </button>
          <p className="auth-alt">New customer? <Link to="/signup">Create an account</Link></p>
          <Tip title="Sign-in limit">
            After several wrong attempts in a minute the bank pauses sign-ins for a short while. Wait a moment, then try again.
          </Tip>
        </form>
      </section>
    </div>
  )
}
