import { Link } from 'react-router-dom'
import { get } from '../../api'
import { useAuth } from '../../auth.jsx'
import { Alert, Card, Chip, Empty, Loading, PageHeader, Tip, useLoad } from '../../components/ui.jsx'
import { dateTime, money, titleCase } from '../../format'

// Loads the customer's accounts, then the 5 latest transactions of each to build "recent activity".
async function loadOverview() {
  const acc = await get('/accounts')
  const accounts = acc.data || []
  const lists = await Promise.all(
    accounts.map((a) =>
      get(`/accounts/${a.id}/transactions`, { pageSize: 5, sort: 'createdAt', direction: 'DESC' })
        .then((r) => (r.data || []).map((t) => ({ ...t, account: a })))
        .catch(() => []),
    ),
  )
  const recent = lists.flat().sort((x, y) => (x.createdAt < y.createdAt ? 1 : -1)).slice(0, 6)
  return { data: { accounts, recent }, meta: acc.meta }
}

export default function Dashboard() {
  const { user } = useAuth()
  const { loading, data, error } = useLoad(loadOverview, [])

  const total = data ? data.accounts.filter((a) => a.currency === 'INR').reduce((s, a) => s + Number(a.balance), 0) : 0

  return (
    <>
      <PageHeader
        title={`Welcome, ${user.username}`}
        subtitle="Your accounts and latest activity."
        actions={<Link className="btn btn-primary" to="/send">Send money</Link>}
      />
      <Alert error={error} />
      {loading ? (
        <Loading />
      ) : data ? (
        <>
          <div className="grid-summary">
            <div className="stat stat-primary">
              <span className="stat-label">Total balance</span>
              <span className="stat-value">{money(total.toFixed(2))}</span>
              <span className="stat-foot">Across {data.accounts.length} {data.accounts.length === 1 ? 'account' : 'accounts'}</span>
            </div>
            <Tip title="Reading your balance">
              The balance shown is what the bank holds for you right now. A transfer reduces it straight away, and the amount shows in the statement with the balance after it.
            </Tip>
          </div>

          <h2 className="section-title">Accounts</h2>
          {data.accounts.length === 0 ? (
            <Card><Empty title="No accounts linked yet">A new profile starts without an account. Ask the bank to link one to your profile.</Empty></Card>
          ) : (
            <div className="account-grid">
              {data.accounts.map((a) => (
                <Link key={a.id} to={`/accounts/${a.id}`} className="account-card">
                  <div className="account-top">
                    <span className="account-type">{titleCase(a.accountType)} account</span>
                    <Chip value={a.status} />
                  </div>
                  <div className="account-number">{a.accountNumberMasked}</div>
                  <div className="account-balance">{money(a.balance)}</div>
                  <div className="account-foot">View statement</div>
                </Link>
              ))}
            </div>
          )}

          <h2 className="section-title">Recent activity</h2>
          <Card flush>
            {data.recent.length === 0 ? (
              <Empty title="No activity yet">Transactions appear here as soon as money moves.</Empty>
            ) : (
              <div className="table-wrap">
                <table className="table">
                  <thead><tr><th>Date</th><th>Description</th><th>Account</th><th className="num">Amount</th></tr></thead>
                  <tbody>
                    {data.recent.map((t) => (
                      <tr key={t.id}>
                        <td className="nowrap">{dateTime(t.createdAt)}</td>
                        <td>{t.description || '—'}</td>
                        <td>{t.account.accountNumberMasked}</td>
                        <td className={'num strong ' + (t.type === 'CREDIT' ? 'pos' : '')}>{(t.type === 'CREDIT' ? '+ ' : '− ') + money(t.amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </>
      ) : null}
    </>
  )
}
