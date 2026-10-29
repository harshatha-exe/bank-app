import { NavLink, Navigate, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth.jsx'
import { CopyButton, Loading, Logo } from './ui.jsx'
import { titleCase } from '../format'

const NAV = {
  CUSTOMER: [
    { to: '/', label: 'Overview', end: true },
    { to: '/send', label: 'Send money' },
    { to: '/transfers', label: 'Transfers' },
    { to: '/beneficiaries', label: 'Beneficiaries' },
  ],
  OPERATIONS: [
    { to: '/ops/accounts', label: 'Accounts' },
    { to: '/ops/transfers', label: 'Transfer review' },
  ],
  ADMIN: [
    { to: '/admin/users', label: 'Users' },
    { to: '/admin/assignments', label: 'Account assignments' },
    { to: '/admin/limit', label: 'Transfer limit' },
    { to: '/admin/audit', label: 'Audit log' },
  ],
}

const AREA = { CUSTOMER: 'Personal banking', OPERATIONS: 'Operations desk', ADMIN: 'Administration' }

export default function Layout() {
  const { user, loading, logout } = useAuth()
  const navigate = useNavigate()
  if (loading) return <Loading label="Checking your session…" />
  if (!user) return <Navigate to="/login" replace />

  async function signOut() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="sidebar-top">
          <Logo light />
          <div className="area">{AREA[user.role]}</div>
        </div>
        <nav aria-label="Main">
          {NAV[user.role].map((n) => (
            <NavLink key={n.to} to={n.to} end={n.end} className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-foot flex-wrap">
          <div className="who">
            <span className="avatar" aria-hidden="true">{user.username.slice(0, 1).toUpperCase()}</span>
            <span>
              <strong>{user.username}</strong>
              <small>{titleCase(user.role)}</small>
            </span>
          </div>
          <div className="order-last w-full rounded-md border border-white/15 px-3 py-2 text-xs text-slate-300 md:order-none">
            <div className="mb-1 uppercase tracking-wider text-slate-400">Your user ID</div>
            <code className="block break-all bg-transparent p-0 text-[11px] text-slate-100" title="Share this with an administrator when they need to assign accounts or change your role">{user.id}</code>
            <div className="mt-1 [&_button]:text-brand-glow"><CopyButton text={user.id} label="Copy ID" /></div>
          </div>
          <button className="btn btn-ghost-light" onClick={signOut}>Sign out</button>
        </div>
      </aside>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
