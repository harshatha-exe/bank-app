import { Link, useParams } from 'react-router-dom'
import { PageHeader, Tip } from '../../components/ui.jsx'
import Transactions from '../../components/Transactions.jsx'

export default function OpsAccountDetail() {
  const { id } = useParams()
  return (
    <>
      <PageHeader title="Account transactions" subtitle={<Link to="/ops/accounts">← All assigned accounts</Link>} />
      <Transactions basePath={`/operations/accounts/${id}`} />
      <Tip title="Access">You can only open accounts an administrator has assigned to you. Anything else shows as not found.</Tip>
    </>
  )
}
