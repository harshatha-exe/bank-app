import { useState } from 'react'
import { get, patch } from '../../api'
import { Alert, Card, Chip, Empty, Field, IdCell, Loading, PageHeader, Pager, Tip, useLoad } from '../../components/ui.jsx'
import { dateTime, money, toRfc3339 } from '../../format'

const EMPTY = { reference: '', status: '', reviewStatus: '', dateFrom: '', dateTo: '' }

export default function OpsTransfers() {
  const [draft, setDraft] = useState(EMPTY)
  const [applied, setApplied] = useState(EMPTY)
  const [page, setPage] = useState(1)
  const [rowBusy, setRowBusy] = useState('')
  const [actionError, setActionError] = useState(null)
  const { loading, data, meta, error, reload } = useLoad(
    () => get('/operations/transfers', { page, pageSize: 10, reference: applied.reference, status: applied.status, reviewStatus: applied.reviewStatus, dateFrom: toRfc3339(applied.dateFrom), dateTo: toRfc3339(applied.dateTo, true) }),
    [page, applied],
  )

  async function review(ref, reviewStatus) {
    setRowBusy(ref)
    setActionError(null)
    try {
      await patch(`/operations/transfers/${ref}/review`, { reviewStatus })
      reload()
    } catch (e) {
      setActionError(e)
    }
    setRowBusy('')
  }

  const set = (k) => (e) => setDraft({ ...draft, [k]: e.target.value })
  return (
    <>
      <PageHeader title="Transfer review" subtitle="Transfers on accounts assigned to you." />
      <Alert error={error} />
      <Alert error={actionError} />
      <Card flush>
        <form className="filters five" onSubmit={(e) => { e.preventDefault(); setPage(1); setApplied(draft) }}>
          <Field label="Reference" htmlFor="f-ref"><input id="f-ref" value={draft.reference} onChange={set('reference')} placeholder="TRF-…" /></Field>
          <Field label="Status" htmlFor="f-st">
            <select id="f-st" value={draft.status} onChange={set('status')}>
              <option value="">Any</option><option value="COMPLETED">Completed</option><option value="PENDING">Pending</option><option value="FAILED">Failed</option>
            </select>
          </Field>
          <Field label="Review" htmlFor="f-rv">
            <select id="f-rv" value={draft.reviewStatus} onChange={set('reviewStatus')}>
              <option value="">Any</option><option value="NONE">None</option><option value="UNDER_REVIEW">Under review</option><option value="RESOLVED">Resolved</option>
            </select>
          </Field>
          <Field label="From date" htmlFor="f-df"><input id="f-df" type="date" value={draft.dateFrom} onChange={set('dateFrom')} /></Field>
          <Field label="To date" htmlFor="f-dt"><input id="f-dt" type="date" value={draft.dateTo} onChange={set('dateTo')} /></Field>
          <div className="filter-actions">
            <button className="btn btn-primary">Apply</button>
            <button type="button" className="btn btn-quiet" onClick={() => { setDraft(EMPTY); setApplied(EMPTY); setPage(1) }}>Clear</button>
          </div>
        </form>
        {loading ? (
          <Loading />
        ) : data && data.length === 0 ? (
          <Empty title="No transfers found">Nothing matches these filters, or no accounts are assigned to you yet.</Empty>
        ) : (
          data && (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Reference</th><th>From / To</th><th className="num">Amount</th><th>Status</th><th>Review</th></tr></thead>
                <tbody>
                  {data.map((t) => (
                    <tr key={t.referenceNumber}>
                      <td><div className="strong nowrap">{t.referenceNumber}</div><div className="subtle">{dateTime(t.createdAt)}</div></td>
                      <td><div className="id-line">From <IdCell id={t.sourceAccountId} /></div><div className="id-line">To {t.destinationAccountMasked || <IdCell id={t.destinationAccountId} />}</div></td>
                      <td className="num strong">{money(t.amount)}</td>
                      <td><Chip value={t.status} /></td>
                      <td>
                        <div className="review-cell">
                          <Chip value={t.reviewStatus} />
                          {t.reviewStatus !== 'UNDER_REVIEW' && <button className="btn btn-quiet btn-sm" disabled={rowBusy === t.referenceNumber} onClick={() => review(t.referenceNumber, 'UNDER_REVIEW')}>Mark for review</button>}
                          {t.reviewStatus === 'UNDER_REVIEW' && <button className="btn btn-primary btn-sm" disabled={rowBusy === t.referenceNumber} onClick={() => review(t.referenceNumber, 'RESOLVED')}>Resolve</button>}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
        <Pager meta={meta} onPage={setPage} />
      </Card>
      <Tip title="Review flow">Mark a transfer for review while you look into it, then resolve it when you are done. Each change is written to the audit log.</Tip>
    </>
  )
}
