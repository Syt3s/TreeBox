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
        "rounded-full text-slate-600 hover:text-slate-950 dark:text-slate-300 dark:hover:text-white",
        active && "bg-slate-100 text-slate-950 dark:bg-slate-800 dark:text-white"
      )}
    >
      <Link href={href}>
        <span className="inline-flex items-center gap-1.5">
          <span>{label}</span>
          {showIndicator ? (
            <span className="inline-flex h-2.5 w-2.5 rounded-full bg-cyan-500 ring-2 ring-white dark:ring-slate-950" />
          ) : null}
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
      // Keep the user on the current page if logout failed on the server.
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
          href: "/user/workspaces",
          label: "工作区",
          active: pathname.startsWith("/user/workspaces"),
        },
        {
          href: "/user/teams",
          label: "团队成员",
          active: pathname.startsWith("/user/teams"),
        },
        {
          href: "/user/profile",
          label: "设置",
          active: pathname.startsWith("/user/profile"),
        },
      ]
    : []

  return (
    <nav className="sticky top-0 z-40 w-full border-b border-slate-200 bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/70 dark:border-slate-800 dark:bg-slate-950/90">
      <div className="container mx-auto flex min-h-16 flex-wrap items-center justify-between gap-3 px-4 py-3">
        <Link href="/" className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-br from-sky-500 via-cyan-500 to-emerald-400 shadow-lg shadow-cyan-500/20">
            <span className="text-lg font-bold text-white">T</span>
          </div>
          <div className="space-y-0.5">
            <span className="block text-lg font-semibold text-slate-950 dark:text-white">TreeBox</span>
            <span className="block text-xs text-slate-500 dark:text-slate-400">多租户问答工作台</span>
          </div>
        </Link>

        <div className="flex flex-wrap items-center justify-end gap-3">
          {loading ? (
            <div className="h-9 w-24 animate-pulse rounded-full bg-slate-200 dark:bg-slate-800" />
          ) : user ? (
            <>
              <div className="flex flex-wrap items-center gap-1 rounded-full border border-slate-200 bg-slate-50/80 p-1 dark:border-slate-800 dark:bg-slate-900/80">
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

              <div className="flex items-center gap-3 rounded-full border border-slate-200 bg-white px-3 py-1.5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
                <Avatar>
                  <AvatarImage src={user.avatar} alt={user.name} />
                  <AvatarFallback>{user.name.charAt(0).toUpperCase()}</AvatarFallback>
                </Avatar>
                <div className="hidden min-w-0 sm:block">
                  <p className="truncate text-sm font-medium text-slate-900 dark:text-slate-100">{user.name}</p>
                  <p className="truncate text-xs text-slate-500 dark:text-slate-400">@{user.domain}</p>
                </div>
                <Button variant="outline" size="sm" onClick={handleLogout}>
                  退出登录
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
