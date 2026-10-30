import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import Beneficiaries from './Beneficiaries'
import { get, post } from '../../api'

vi.mock('../../api', () => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  friendly: (error) => error?.message || 'Something went wrong.',
}))

describe('Beneficiaries', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    get.mockResolvedValue({ data: [{ id: 'b1', nickname: '<img src=x onerror=alert(1)>', destinationAccountMasked: '******3456', status: 'ACTIVE' }] })
  })

  it('renders masked destinations and treats nicknames as text', async () => {
    render(<MemoryRouter><Beneficiaries /></MemoryRouter>)
    expect(await screen.findByText('<img src=x onerror=alert(1)>')).toBeInTheDocument()
    expect(screen.getByText('******3456')).toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('shows a friendly add error', async () => {
    post.mockRejectedValueOnce({ message: 'You cannot add your own account.' })
    render(<MemoryRouter><Beneficiaries /></MemoryRouter>)
    await screen.findByText('******3456')
    fireEvent.change(screen.getByLabelText('Account ID'), { target: { value: '00000000-0000-0000-0000-000000000001' } })
    fireEvent.change(screen.getByLabelText('Nickname'), { target: { value: 'Savings' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add beneficiary' }))
    await waitFor(() => expect(screen.getByText('You cannot add your own account.')).toBeInTheDocument())
  })
})
