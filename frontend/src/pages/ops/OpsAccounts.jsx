import { useState } from 'react'
import { Link } from 'react-router-dom'
import { get, post } from '../../api'
import { Alert, Card, Chip, ConfirmDialog, Empty, IdCell, Loading, PageHeader, Tip, useLoad } from '../../components/ui.jsx'
import { money, titleCase } from '../../format'

export default function OpsAccounts() {
  const { loading, data, error, reload } = useLoad(() => get('/operations/accounts'), [])
  const [target, setTarget] = useState(null)
  const [busy, setBusy] = useState(false)
  const [actionError, setActionError] = useState(null)

  async function run() {
    setBusy(true)
    setActionError(null)
    try {
      await post(`/operations/accounts/${target.id}/${target.status === 'FROZEN' ? 'unfreeze' : 'freeze'}`)
      setTarget(null)
      reload()
    } catch (e) {
      setActionError(e)
    }
    setBusy(false)
  }

  const freezing = target && target.status !== 'FROZEN'
  return (
    <>
      <PageHeader title="Accounts" subtitle="Accounts assigned to you by an administrator." />
      <Alert error={error} />
      <Card flush>
        {loading ? (
          <Loading />
        ) : data && data.length === 0 ? (
          <Empty title="No accounts assigned yet">An administrator assigns accounts to operations staff. Once that is done, they appear here.</Empty>
        ) : (
          data && (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Account</th><th>Type</th><th>Account ID</th><th className="num">Balance</th><th>Status</th><th></th></tr></thead>
                <tbody>
                  {data.map((a) => (
                    <tr key={a.id}>
                      <td><Link to={`/ops/accounts/${a.id}`} className="strong">{a.accountNumberMasked}</Link></td>
                      <td>{titleCase(a.accountType)}</td>
                      <td><IdCell id={a.id} /></td>
                      <td className="num strong">{money(a.balance)}</td>
                      <td><Chip value={a.status} /></td>
                      <td className="num">
                        <button className="btn btn-quiet btn-sm" onClick={() => { setActionError(null); setTarget(a) }}>
                          {a.status === 'FROZEN' ? 'Unfreeze' : 'Freeze'}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
      </Card>
      <Tip title="About freezing">A frozen account cannot send or receive transfers. Freezing and unfreezing are recorded in the audit log.</Tip>
      <ConfirmDialog open={!!target} title={freezing ? 'Freeze this account?' : 'Unfreeze this account?'} confirmLabel={freezing ? 'Freeze account' : 'Unfreeze account'} busy={busy} error={actionError} onConfirm={run} onCancel={() => setTarget(null)}>
        {target && <p>{titleCase(target.accountType)} account {target.accountNumberMasked}. {freezing ? 'No money will be able to move in or out until it is unfrozen.' : 'Transfers will be allowed again.'}</p>}
      </ConfirmDialog>
    </>
  )
}
