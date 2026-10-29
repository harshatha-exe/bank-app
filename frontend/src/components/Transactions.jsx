import { useState } from 'react'
import { get } from '../api'
import { dateTime, money, MONEY_RE, toRfc3339 } from '../format'
import { Alert, Card, Chip, Empty, Loading, Pager, useLoad } from './ui.jsx'

const EMPTY = { type: '', from: '', to: '', minAmount: '', maxAmount: '', sort: 'createdAt', direction: 'DESC' }

// Statement table with the filters the backend supports. basePath is
// /accounts/<id> for customers and /operations/accounts/<id> for operations staff.
export default function Transactions({ basePath }) {
  const [draft, setDraft] = useState(EMPTY)
  const [applied, setApplied] = useState(EMPTY)
  const [page, setPage] = useState(1)
  const [localError, setLocalError] = useState('')

  const { loading, data, meta, error } = useLoad(
    () =>
      get(basePath + '/transactions', {
        page,
        pageSize: 10,
        type: applied.type,
        sort: applied.sort,
        direction: applied.direction,
        from: toRfc3339(applied.from),
        to: toRfc3339(applied.to, true),
        minAmount: applied.minAmount,
        maxAmount: applied.maxAmount,
      }),
    [basePath, page, applied],
  )

  function apply(e) {
    e.preventDefault()
    for (const k of ['minAmount', 'maxAmount']) {
      if (draft[k] && !MONEY_RE.test(draft[k])) {
        setLocalError('Amounts need digits with at most two decimals, for example 2500 or 2500.50.')
        return
      }
    }
    setLocalError('')
    setPage(1)
    setApplied(draft)
  }
  function reset() {
    setDraft(EMPTY)
    setApplied(EMPTY)
    setPage(1)
    setLocalError('')
  }
  const set = (k) => (e) => setDraft({ ...draft, [k]: e.target.value })

  return (
    <Card title="Transactions" flush>
      <form className="filters" onSubmit={apply}>
        <label>Type
          <select value={draft.type} onChange={set('type')}>
            <option value="">All</option>
            <option value="DEBIT">Money out (debit)</option>
            <option value="CREDIT">Money in (credit)</option>
          </select>
        </label>
        <label>From
          <input type="date" value={draft.from} onChange={set('from')} />
        </label>
        <label>To
          <input type="date" value={draft.to} onChange={set('to')} />
        </label>
        <label>Min amount
          <input inputMode="decimal" placeholder="0" value={draft.minAmount} onChange={set('minAmount')} />
        </label>
        <label>Max amount
          <input inputMode="decimal" placeholder="Any" value={draft.maxAmount} onChange={set('maxAmount')} />
        </label>
        <label>Sort by
          <select value={draft.sort} onChange={set('sort')}>
            <option value="createdAt">Date</option>
            <option value="amount">Amount</option>
            <option value="type">Type</option>
          </select>
        </label>
        <label>Order
          <select value={draft.direction} onChange={set('direction')}>
            <option value="DESC">Newest / highest first</option>
            <option value="ASC">Oldest / lowest first</option>
          </select>
        </label>
        <div className="filter-actions">
          <button className="btn btn-primary">Apply</button>
          <button type="button" className="btn btn-quiet" onClick={reset}>Reset</button>
        </div>
      </form>
      {localError && <div className="pad"><Alert>{localError}</Alert></div>}
      {error && <div className="pad"><Alert error={error} /></div>}
      {loading ? (
        <Loading />
      ) : data && data.length === 0 ? (
        <Empty title="No transactions found">Nothing matches these filters yet.</Empty>
      ) : (
        data && (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Description</th>
                  <th>Type</th>
                  <th className="num">Amount</th>
                  <th className="num">Balance after</th>
                </tr>
              </thead>
              <tbody>
                {data.map((t) => (
                  <tr key={t.id}>
                    <td className="nowrap">{dateTime(t.createdAt)}</td>
                    <td>{t.description || '—'}</td>
                    <td><Chip value={t.type} label={t.type === 'CREDIT' ? 'Money in' : 'Money out'} /></td>
                    <td className={'num strong ' + (t.type === 'CREDIT' ? 'pos' : '')}>{(t.type === 'CREDIT' ? '+ ' : '− ') + money(t.amount)}</td>
                    <td className="num">{money(t.balanceAfter)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}
      <Pager meta={meta} onPage={setPage} />
    </Card>
  )
}
