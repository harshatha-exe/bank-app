import { Link, useParams } from 'react-router-dom'
import { get } from '../../api'
import { Alert, Card, Chip, CopyButton, Loading, PageHeader, Tip, useLoad } from '../../components/ui.jsx'
import { dateTime, money } from '../../format'
import { accountLabel, destinationLabel } from './Transfers.jsx'

async function load(ref) {
  const [t, a, b] = await Promise.all([get(`/transfers/${ref}`), get('/accounts'), get('/beneficiaries')])
  return { data: { t: t.data, accounts: a.data || [], beneficiaries: b.data || [] }, meta: null }
}

export default function TransferDetail() {
  const { ref } = useParams()
  const { loading, data, error } = useLoad(() => load(ref), [ref])
  const t = data && data.t
  return (
    <>
      <PageHeader title="Transfer details" subtitle={<Link to="/transfers">← All transfers</Link>} />
      <Alert error={error} />
      {loading ? (
        <Loading />
      ) : t ? (
        <div className="two-col">
          <Card title={t.referenceNumber} aside={<Chip value={t.status} />}>
            <dl className="summary wide">
              <dt>Amount</dt><dd className="strong big">{money(t.amount)}</dd>
              <dt>From</dt><dd>{accountLabel(t.sourceAccountId, data.accounts, data.beneficiaries)}</dd>
              <dt>To</dt><dd>{destinationLabel(t, data.accounts, data.beneficiaries)}</dd>
              <dt>Review</dt><dd><Chip value={t.reviewStatus} /></dd>
              <dt>Purpose</dt><dd>{t.purpose || '—'}</dd>
              <dt>Date</dt><dd>{dateTime(t.createdAt)}</dd>
              <dt>Reference</dt><dd>{t.referenceNumber} <CopyButton text={t.referenceNumber} /></dd>
            </dl>
          </Card>
          <Tip title="What the status means">
            Completed means the money has moved and both statements show it. Pending and failed transfers show here too, so you always have a record.
          </Tip>
        </div>
      ) : null}
    </>
  )
}
