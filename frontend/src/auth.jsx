import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { get, post, setCsrf } from './api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  // On first load (and on page refresh) ask the backend who we are. If the session cookie is
  // still valid, /me also hands back a fresh CSRF token.
  useEffect(() => {
    let live = true
    get('/me')
      .then((res) => live && setUser(res.data))
      .catch(() => live && setUser(null))
      .finally(() => live && setLoading(false))
    return () => {
      live = false
    }
  }, [])

  const login = useCallback(async (username, password) => {
    const res = await post('/auth/login', { username, password })
    setCsrf(res.meta.csrfToken)
    setUser(res.data.user)
    return res.data.user
  }, [])

  const logout = useCallback(async () => {
    try {
      await post('/auth/logout')
    } catch {
      // Even if the call fails, drop local state so the user lands on the sign-in page.
    }
    setCsrf(null)
    setUser(null)
  }, [])

  return <AuthContext.Provider value={{ user, loading, login, logout, setUser }}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)

export function homePathFor(role) {
  if (role === 'ADMIN') return '/admin/users'
  if (role === 'OPERATIONS') return '/ops/accounts'
  return '/'
}
