"use client"

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react"
import { api } from "@/lib/api"
import type { User } from "@/types"

type AuthContextValue = {
  user: User | null
  loading: boolean
  refreshUser: () => Promise<User | null>
  setUser: (user: User | null) => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUserState] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const refreshRequestIdRef = useRef(0)

  const setUser = useCallback((nextUser: User | null) => {
    refreshRequestIdRef.current += 1
    setUserState(nextUser)
    setLoading(false)
  }, [])

  const refreshUser = useCallback(async () => {
    const requestId = ++refreshRequestIdRef.current

    try {
      const currentUser = await api.auth.getCurrentUser()
      if (requestId !== refreshRequestIdRef.current) {
        return currentUser
      }
      setUserState(currentUser)
      return currentUser
    } catch {
      if (requestId !== refreshRequestIdRef.current) {
        return null
      }
      setUserState(null)
      return null
    } finally {
      if (requestId === refreshRequestIdRef.current) {
        setLoading(false)
      }
    }
  }, [])

  const logout = useCallback(async () => {
    refreshRequestIdRef.current += 1
    await api.auth.logout()
    setUserState(null)
    setLoading(false)
  }, [])

  useEffect(() => {
    void refreshUser()
  }, [refreshUser])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      loading,
      refreshUser,
      setUser,
      logout,
    }),
    [user, loading, refreshUser, setUser, logout]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider")
  }
  return context
}