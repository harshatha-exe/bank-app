import { Link, useParams } from 'react-router-dom'
import { get } from '../../api'
import { Alert, Card, Chip, CopyButton, Loading, PageHeader, Tip, useLoad } from '../../components/ui.jsx'
import Transactions from '../../components/Transactions.jsx'
import { money, titleCase } from '../../format'

export default function AccountDetail() {
  const { id } = useParams()
  const { loading, data, error } = useLoad(() => get(`/accounts/${id}`), [id])

  return (
    <>
      <PageHeader title="Account statement" subtitle={<Link to="/">← Back to overview</Link>} />
      <Alert error={error} />
      {loading ? (
        <Loading />
      ) : data ? (
        <>
          <div className="grid-summary">
            <div className="stat stat-primary">
              <span className="stat-label">{titleCase(data.accountType)} account · {data.accountNumberMasked}</span>
              <span className="stat-value">{money(data.balance)}</span>
              <span className="stat-foot"><Chip value={data.status} /></span>
            </div>
            <Tip title="Your account ID">
              Someone who wants to add this account as a beneficiary needs its account ID, not the masked number.
              <div className="tip-id"><code>{data.id}</code> <CopyButton text={data.id} label="Copy ID" /></div>
            </Tip>
          </div>
          {data.status === 'FROZEN' && (
            <Alert kind="warn">This account is frozen. Money cannot be sent from or to it until the bank unfreezes it.</Alert>
          )}
          <Transactions basePath={`/accounts/${id}`} />
        </>
      ) : null}
    </>
  )
}
