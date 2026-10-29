import { Fragment, useState } from 'react'
import { get } from '../../api'
import { Alert, Card, Chip, Empty, Field, Loading, PageHeader, Pager, Tip, useLoad } from '../../components/ui.jsx'
import { dateTime, shortId, toRfc3339 } from '../../format'

const EMPTY = { eventType: '', actorUserId: '', entityType: '', entityId: '', result: '', from: '', to: '' }

export default function AdminAudit() {
  const [draft, setDraft] = useState(EMPTY)
  const [applied, setApplied] = useState(EMPTY)
  const [page, setPage] = useState(1)
  const [open, setOpen] = useState('')
  const { loading, data, meta, error } = useLoad(
    () => get('/admin/audit-events', { page, pageSize: 15, eventType: applied.eventType, actorUserId: applied.actorUserId, entityType: applied.entityType, entityId: applied.entityId, result: applied.result, from: toRfc3339(applied.from), to: toRfc3339(applied.to, true) }),
    [page, applied],
  )
  const set = (k) => (e) => setDraft({ ...draft, [k]: e.target.value.trim() })
  return (
    <>
      <PageHeader title="Audit log" subtitle="A record of sensitive actions across the bank." />
      <Alert error={error} />
      <Card flush>
        <form className="filters" onSubmit={(e) => { e.preventDefault(); setPage(1); setApplied(draft) }}>
          <Field label="Event type" htmlFor="au-et"><input id="au-et" value={draft.eventType} onChange={set('eventType')} placeholder="e.g. LOGIN_SUCCESS" /></Field>
          <Field label="Result" htmlFor="au-r">
            <select id="au-r" value={draft.result} onChange={set('result')}><option value="">Any</option><option value="SUCCESS">Success</option><option value="FAILURE">Failure</option></select>
          </Field>
          <Field label="Actor user ID" htmlFor="au-a"><input id="au-a" value={draft.actorUserId} onChange={set('actorUserId')} /></Field>
          <Field label="Entity type" htmlFor="au-t"><input id="au-t" value={draft.entityType} onChange={set('entityType')} /></Field>
          <Field label="Entity ID" htmlFor="au-e"><input id="au-e" value={draft.entityId} onChange={set('entityId')} /></Field>
          <Field label="From" htmlFor="au-f"><input id="au-f" type="date" value={draft.from} onChange={set('from')} /></Field>
          <Field label="To" htmlFor="au-to"><input id="au-to" type="date" value={draft.to} onChange={set('to')} /></Field>
          <div className="filter-actions">
            <button className="btn btn-primary">Apply</button>
            <button type="button" className="btn btn-quiet" onClick={() => { setDraft(EMPTY); setApplied(EMPTY); setPage(1) }}>Clear</button>
          </div>
        </form>
        {loading ? <Loading /> : data && data.length === 0 ? <Empty title="No events match" /> : data && (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>Time</th><th>Event</th><th>Action</th><th>Result</th><th>Actor</th><th>IP</th><th></th></tr></thead>
              <tbody>
                {data.map((ev) => (
                  <Fragment key={ev.id}>
                    <tr>
                      <td className="nowrap">{dateTime(ev.createdAt)}</td>
                      <td className="strong">{ev.eventType}</td>
                      <td>{ev.action}</td>
                      <td><Chip value={ev.result} /></td>
                      <td title={ev.actorUserId || ''}>{ev.actorUserId ? shortId(ev.actorUserId) : 'System'}</td>
                      <td>{ev.ipAddress || '—'}</td>
                      <td className="num"><button className="btn-link" onClick={() => setOpen(open === ev.id ? '' : ev.id)}>{open === ev.id ? 'Hide' : 'Details'}</button></td>
                    </tr>
                    {open === ev.id && (
                      <tr><td colSpan="7">
                        <dl className="summary wide">
                          <dt>Event ID</dt><dd><code>{ev.id}</code></dd>
                          <dt>Correlation</dt><dd><code>{ev.correlationId}</code></dd>
                          {ev.entityType && (<><dt>Entity</dt><dd>{ev.entityType} <code>{ev.entityId}</code></dd></>)}
                          <dt>Metadata</dt><dd><code>{ev.metadata ? JSON.stringify(ev.metadata) : '—'}</code></dd>
                        </dl>
                      </td></tr>
                    )}
                  </Fragment>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <Pager meta={meta} onPage={setPage} />
      </Card>
      <Tip title="Correlation ID">Every error message in this app shows a reference. Search the log by that value to find the matching event.</Tip>
    </>
  )
}
