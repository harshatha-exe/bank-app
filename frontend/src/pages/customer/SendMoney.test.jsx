import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import SendMoney from './SendMoney'
import { get, post } from '../../api'

vi.mock('../../api', () => ({
  get: vi.fn(),
  post: vi.fn(),
  friendly: (error) => error?.message || 'Something went wrong.',
}))

const accounts = [
  { id: 'account-1', accountType: 'CURRENT', accountNumberMasked: '****1111', balance: '5000.00', status: 'ACTIVE' },
  { id: 'account-2', accountType: 'SAVINGS', accountNumberMasked: '****2222', balance: '2500.50', status: 'ACTIVE' },
  { id: 'account-3', accountType: 'SAVINGS', accountNumberMasked: '****3333', balance: '100.00', status: 'FROZEN' },
]
const beneficiaries = [{ id: 'beneficiary-1', nickname: 'Rent', destinationAccountId: 'external-1', destinationAccountMasked: '****9999', status: 'ACTIVE' }]

function renderPage(data = { accounts, beneficiaries }) {
  get.mockResolvedValueOnce({ data: data.accounts || accounts }).mockResolvedValueOnce({ data: data.beneficiaries || beneficiaries })
  post.mockResolvedValue({ data: { amount: '10.00', referenceNumber: 'REF-1', status: 'COMPLETED' } })
  return render(<MemoryRouter><SendMoney /></MemoryRouter>)
}

async function loaded() {
  await screen.findByLabelText('From account')
}

function chooseTransfer(source, destination, amount = '10') {
  fireEvent.change(screen.getByLabelText('From account'), { target: { value: source } })
  fireEvent.change(screen.getByLabelText(/To (account|beneficiary)/), { target: { value: destination } })
  fireEvent.change(screen.getByLabelText('Amount (INR)'), { target: { value: amount } })
}

describe('SendMoney destination modes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
  })

  it('renders both destination modes', async () => {
    renderPage()
    await loaded()

    expect(screen.getByRole('button', { name: 'My accounts' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Saved beneficiary' })).toBeInTheDocument()
    expect(screen.getByLabelText('To beneficiary')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'My accounts' }))
    expect(screen.getByLabelText('To account')).toBeInTheDocument()
  })

  it('excludes the chosen source from my-account destinations', async () => {
    renderPage()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: 'My accounts' }))
    fireEvent.change(screen.getByLabelText('From account'), { target: { value: 'account-1' } })

    const options = [...screen.getByLabelText('To account').options]
    expect(options.map((option) => option.value)).not.toContain('account-1')
    expect(options.map((option) => option.value)).toContain('account-2')
  })

  it('submits the correct body for both destination modes', async () => {
    renderPage()
    await loaded()
    chooseTransfer('account-1', 'beneficiary-1')
    fireEvent.click(screen.getByRole('button', { name: /Review transfer/ }))
    fireEvent.click(screen.getByRole('button', { name: /Send/ }))
    await waitFor(() => expect(post).toHaveBeenCalledWith('/transfers', expect.objectContaining({ sourceAccountId: 'account-1', destinationAccountId: 'external-1', beneficiaryId: 'beneficiary-1', amount: '10', purpose: '' }), expect.anything()))

    renderPage()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: 'My accounts' }))
    chooseTransfer('account-1', 'account-2')
    fireEvent.click(screen.getByRole('button', { name: /Review transfer/ }))
    fireEvent.click(screen.getByRole('button', { name: /Send/ }))
    await waitFor(() => expect(post).toHaveBeenLastCalledWith('/transfers', { sourceAccountId: 'account-1', destinationAccountId: 'account-2', amount: '10', purpose: '' }, expect.anything()))
  })

  it('clears the destination when the source changes to it', async () => {
    renderPage()
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: 'My accounts' }))
    chooseTransfer('account-1', 'account-2')
    fireEvent.change(screen.getByLabelText('From account'), { target: { value: 'account-2' } })
    expect(screen.getByLabelText('To account')).toHaveValue('')
  })

  it('disables frozen accounts in both selects', async () => {
    renderPage()
    await loaded()
    expect(screen.getByLabelText('From account').querySelector('option[value="account-3"]')).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'My accounts' }))
    expect(screen.getByLabelText('To account').querySelector('option[value="account-3"]')).toBeDisabled()
  })
})
