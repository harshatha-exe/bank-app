import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import AdminUsers from './AdminUsers'
import { get, patch, post } from '../../api'

vi.mock('../../api', () => ({
  get: vi.fn(),
  patch: vi.fn(),
  post: vi.fn(),
  friendly: (error) => error?.message || 'Something went wrong.',
}))

const users = [
  { id: '10000000-0000-0000-0000-000000000001', username: 'alice', role: 'CUSTOMER', status: 'ACTIVE', createdAt: '2026-10-30T12:00:00Z' },
  { id: '10000000-0000-0000-0000-000000000002', username: 'ops-user', role: 'OPERATIONS', status: 'DEACTIVATED', createdAt: '2026-10-30T12:00:00Z' },
]
const meta = { page: 1, pageSize: 15, totalItems: 2, totalPages: 1 }

function renderPage(data = users) {
  get.mockResolvedValue({ data, meta })
  patch.mockResolvedValue({ data: {} })
  post.mockResolvedValue({ data: {} })
  return render(<MemoryRouter><AdminUsers /></MemoryRouter>)
}

async function loaded() {
  await screen.findByText('alice')
}

describe('AdminUsers', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loaded user rows', async () => {
    renderPage()
    await loaded()
    expect(screen.getAllByText('Customer').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Active').length).toBeGreaterThan(0)
    expect(screen.getByText('ops-user')).toBeInTheDocument()
    expect(screen.getAllByText('Operations').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Deactivated').length).toBeGreaterThan(0)
  })

  it('applies filters and resets to page one', async () => {
    renderPage()
    await loaded()
    fireEvent.change(screen.getByLabelText('Find username'), { target: { value: ' alice ' } })
    fireEvent.change(screen.getByLabelText('Find role'), { target: { value: 'CUSTOMER' } })
    fireEvent.change(screen.getByLabelText('Find status'), { target: { value: 'ACTIVE' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply' }))
    await waitFor(() => expect(get).toHaveBeenLastCalledWith('/admin/users', { search: 'alice', role: 'CUSTOMER', status: 'ACTIVE', page: 1, pageSize: 15 }))
  })

  it('selects a user into the manage form', async () => {
    renderPage()
    await loaded()
    fireEvent.click(screen.getAllByRole('button', { name: 'Select' })[1])
    expect(screen.getByText('Selected user: ops-user')).toBeInTheDocument()
    expect(screen.getByLabelText('User ID')).toHaveValue(users[1].id)
    expect(screen.getByLabelText('Manage role')).toHaveValue('OPERATIONS')
    expect(screen.getByLabelText('Manage status')).toHaveValue('DEACTIVATED')
  })

  it('confirms and sends a status update, then refetches the list', async () => {
    renderPage()
    await loaded()
    fireEvent.click(screen.getAllByRole('button', { name: 'Select' })[0])
    fireEvent.change(screen.getByLabelText('Manage status'), { target: { value: 'DEACTIVATED' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update status' }))
    expect(patch).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    await waitFor(() => expect(patch).toHaveBeenCalledWith(`/admin/users/${users[0].id}/status`, { status: 'DEACTIVATED' }))
    await waitFor(() => expect(get).toHaveBeenCalledTimes(2))
  })

  it('requires confirmation before deactivating', async () => {
    renderPage()
    await loaded()
    fireEvent.click(screen.getAllByRole('button', { name: 'Select' })[0])
    fireEvent.change(screen.getByLabelText('Manage status'), { target: { value: 'DEACTIVATED' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update status' }))
    expect(screen.getByRole('dialog')).toHaveTextContent('alice')
    expect(patch).not.toHaveBeenCalled()
  })

  it('renders API errors and empty results', async () => {
    get.mockRejectedValueOnce(new Error('Could not load users'))
    render(<MemoryRouter><AdminUsers /></MemoryRouter>)
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load users')

    get.mockResolvedValueOnce({ data: [], meta })
    render(<MemoryRouter><AdminUsers /></MemoryRouter>)
    expect(await screen.findByText('No users match')).toBeInTheDocument()
  })
})
