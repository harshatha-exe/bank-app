import { Navigate, Route, Routes } from 'react-router-dom'
import { homePathFor, useAuth } from './auth.jsx'
import Layout from './components/Layout.jsx'
import { Loading } from './components/ui.jsx'
import Login from './pages/Login.jsx'
import Signup from './pages/Signup.jsx'
import Dashboard from './pages/customer/Dashboard.jsx'
import AccountDetail from './pages/customer/AccountDetail.jsx'
import Beneficiaries from './pages/customer/Beneficiaries.jsx'
import SendMoney from './pages/customer/SendMoney.jsx'
import Transfers from './pages/customer/Transfers.jsx'
import TransferDetail from './pages/customer/TransferDetail.jsx'
import OpsAccounts from './pages/ops/OpsAccounts.jsx'
import OpsAccountDetail from './pages/ops/OpsAccountDetail.jsx'
import OpsTransfers from './pages/ops/OpsTransfers.jsx'
import AdminUsers from './pages/admin/AdminUsers.jsx'
import AdminAssignments from './pages/admin/AdminAssignments.jsx'
import AdminLimit from './pages/admin/AdminLimit.jsx'
import AdminAudit from './pages/admin/AdminAudit.jsx'

// Only lets the listed roles through; everyone else goes to their own home page.
function Guard({ roles, children }) {
  const { user, loading } = useAuth()
  if (loading) return <Loading label="Checking your session…" />
  if (!user) return <Navigate to="/login" replace />
  if (!roles.includes(user.role)) return <Navigate to={homePathFor(user.role)} replace />
  return children
}

function Entry({ children }) {
  const { user, loading } = useAuth()
  if (loading) return <Loading label="Checking your session…" />
  if (user) return <Navigate to={homePathFor(user.role)} replace />
  return children
}

const customer = (el) => <Guard roles={['CUSTOMER']}>{el}</Guard>
const ops = (el) => <Guard roles={['OPERATIONS']}>{el}</Guard>
const admin = (el) => <Guard roles={['ADMIN']}>{el}</Guard>

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Entry><Login /></Entry>} />
      <Route path="/signup" element={<Entry><Signup /></Entry>} />
      <Route element={<Layout />}>
        <Route path="/" element={customer(<Dashboard />)} />
        <Route path="/accounts/:id" element={customer(<AccountDetail />)} />
        <Route path="/beneficiaries" element={customer(<Beneficiaries />)} />
        <Route path="/send" element={customer(<SendMoney />)} />
        <Route path="/transfers" element={customer(<Transfers />)} />
        <Route path="/transfers/:ref" element={customer(<TransferDetail />)} />

        <Route path="/ops/accounts" element={ops(<OpsAccounts />)} />
        <Route path="/ops/accounts/:id" element={ops(<OpsAccountDetail />)} />
        <Route path="/ops/transfers" element={ops(<OpsTransfers />)} />

        <Route path="/admin/users" element={admin(<AdminUsers />)} />
        <Route path="/admin/assignments" element={admin(<AdminAssignments />)} />
        <Route path="/admin/limit" element={admin(<AdminLimit />)} />
        <Route path="/admin/audit" element={admin(<AdminAudit />)} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
