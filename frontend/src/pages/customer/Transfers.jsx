import { useState } from 'react'
import { Link } from 'react-router-dom'
import { get } from '../../api'
import { Alert, Card, Chip, Empty, Loading, PageHeader, Pager, Tip, useLoad } from '../../components/ui.jsx'
import { dateTime, money, shortId } from '../../format'

async function loadAll(page) {
  const [t, a, b] = await Promise.all([get('/transfers', { page, pageSize: 10 }), get('/accounts'), get('/beneficiaries')])
  return { data: { transfers: t.data || [], accounts: a.data || [], beneficiaries: b.data || [] }, meta: t.meta }
}

export function accountLabel(id, accounts, beneficiaries) {
  const own = accounts.find((a) => a.id === id)
  if (own) return `Your ${own.accountNumberMasked}`
  const ben = beneficiaries.find((b) => b.destinationAccountId === id)
  if (ben) return `${ben.nickname} · ${ben.destinationAccountMasked}`
  return `Account ${shortId(id)}`
}

export default function Transfers() {
  const [page, setPage] = useState(1)
  const { loading, data, meta, error } = useLoad(() => loadAll(page), [page])
  return (
    <>
      <PageHeader title="Transfers" subtitle="Transfers you have sent from your accounts." actions={<Link className="btn btn-primary" to="/send">Send money</Link>} />
      <Alert error={error} />
      <Card flush>
        {loading ? (
          <Loading />
        ) : data && data.transfers.length === 0 ? (
          <Empty title="No transfers yet">Transfers you make will be listed here with their reference numbers.</Empty>
        ) : (
          data && (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Reference</th><th>Date</th><th>From</th><th>To</th><th className="num">Amount</th><th>Status</th></tr></thead>
                <tbody>
                  {data.transfers.map((t) => (
                    <tr key={t.referenceNumber}>
                      <td><Link to={`/transfers/${t.referenceNumber}`} className="strong">{t.referenceNumber}</Link></td>
                      <td className="nowrap">{dateTime(t.createdAt)}</td>
                      <td>{accountLabel(t.sourceAccountId, data.accounts, data.beneficiaries)}</td>
                      <td>{accountLabel(t.destinationAccountId, data.accounts, data.beneficiaries)}</td>
                      <td className="num strong">{money(t.amount)}</td>
                      <td><Chip value={t.status} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
        <Pager meta={meta} onPage={setPage} />
      </Card>
      <Tip title="Reference numbers">Quote a transfer's reference number to the bank if you ever need help with it.</Tip>
    </>
  )
}
