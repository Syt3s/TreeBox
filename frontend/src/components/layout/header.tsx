"use client"

import Link from "next/link"
import { useCallback, useEffect, useState } from "react"
import { usePathname, useRouter } from "next/navigation"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/components/auth/auth-provider"
import { api } from "@/lib/api"
import { QUESTION_STATS_REFRESH_EVENT } from "@/lib/question-stats"
import { cn } from "@/lib/utils"

type NavItemProps = {
  href: string
  label: string
  active?: boolean
  showIndicator?: boolean
}

function NavItem({ href, label, active = false, showIndicator = false }: NavItemProps) {
  return (
    <Button
      asChild
      variant="ghost"
      size="sm"
      className={cn(
        "rounded-full text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-gray-50",
        active && "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50"
      )}
    >
      <Link href={href}>
        <span className="inline-flex items-center gap-1.5">
          <span>{label}</span>
          {showIndicator && (
            <span className="inline-flex h-2.5 w-2.5 flex-shrink-0 rounded-full bg-sky-500 ring-2 ring-white dark:bg-cyan-400 dark:ring-gray-950" />
          )}
        </span>
      </Link>
    </Button>
  )
}

export function Header() {
  const { user, logout, loading } = useAuth()
  const router = useRouter()
  const pathname = usePathname()
  const [unreadCount, setUnreadCount] = useState(0)

  const handleLogout = async () => {
    try {
      await logout()
      router.replace("/login")
    } catch {
      // Keep the current page if the server-side logout failed.
    }
  }

  const loadQuestionStats = useCallback(async () => {
    if (!user) {
      setUnreadCount(0)
      return
    }

    try {
      const response = await api.user.questions.stats()
      setUnreadCount(response.unread_count || 0)
    } catch {
      setUnreadCount(0)
    }
  }, [user])

  useEffect(() => {
    void loadQuestionStats()
  }, [loadQuestionStats, pathname])

  useEffect(() => {
    if (typeof window === "undefined") {
      return
    }

    const handleRefresh = () => {
      void loadQuestionStats()
    }

    window.addEventListener(QUESTION_STATS_REFRESH_EVENT, handleRefresh)
    return () => window.removeEventListener(QUESTION_STATS_REFRESH_EVENT, handleRefresh)
  }, [loadQuestionStats])

  const navigation = user
    ? [
        {
          href: `/box/${user.domain}`,
          label: "我的提问箱",
          active: pathname.startsWith(`/box/${user.domain}`),
        },
        {
          href: "/user/questions",
          label: "问题列表",
          active: pathname.startsWith("/user/questions"),
          showIndicator: unreadCount > 0 && !pathname.startsWith("/user/questions"),
        },
        {
          href: "/user/profile",
          label: "设置",
          active: pathname.startsWith("/user/profile"),
        },
      ]
    : []

  return (
    <nav className="sticky top-0 z-40 w-full border-b border-gray-200 bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:border-gray-800 dark:bg-gray-950/95">
      <div className="container mx-auto flex min-h-16 flex-wrap items-center justify-between gap-3 px-4 py-3">
        <Link href="/" className="flex items-center gap-2">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-blue-600 to-cyan-500">
            <span className="text-xl font-bold text-white">T</span>
          </div>
          <span className="text-xl font-bold text-gray-900 dark:text-gray-100">TreeBox</span>
        </Link>

        <div className="flex flex-wrap items-center justify-end gap-3">
          {loading ? (
            <div className="h-9 w-24 animate-pulse rounded-md bg-gray-200 dark:bg-gray-800" />
          ) : user ? (
            <>
              <div className="flex flex-wrap items-center gap-1 rounded-full border border-gray-200/80 bg-gray-50/80 p-1 dark:border-gray-800 dark:bg-gray-900/80">
                {navigation.map((item) => (
                  <NavItem
                    key={item.href}
                    href={item.href}
                    label={item.label}
                    active={item.active}
                    showIndicator={item.showIndicator}
                  />
                ))}
              </div>

              <div className="flex items-center gap-3">
                <Avatar>
                  <AvatarImage src={user.avatar} alt={user.name} />
                  <AvatarFallback>{user.name.charAt(0).toUpperCase()}</AvatarFallback>
                </Avatar>
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{user.name}</span>
                <Button variant="outline" size="sm" onClick={handleLogout}>
                  退出
                </Button>
              </div>
            </>
          ) : (
            <div className="flex items-center gap-2">
              <Button asChild variant="ghost" size="sm">
                <Link href="/login">登录</Link>
              </Button>
              <Button asChild size="sm">
                <Link href="/register">注册</Link>
              </Button>
            </div>
          )}
        </div>
      </div>
    </nav>
  )
}
