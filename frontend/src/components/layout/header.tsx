"use client"

import Link from "next/link"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { useAuth } from "@/components/auth/auth-provider"

export function Header() {
  const { user, logout, loading } = useAuth()
  const router = useRouter()

  const handleLogout = async () => {
    try {
      await logout()
      router.replace("/login")
    } catch {
      // Keep the current page if the server-side logout failed.
    }
  }

  return (
    <nav className="sticky top-0 z-40 w-full border-b border-gray-200 bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:border-gray-800 dark:bg-gray-950/95">
      <div className="container mx-auto flex h-16 items-center justify-between px-4">
        <Link href="/" className="flex items-center gap-2">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-blue-600 to-cyan-500">
            <span className="text-xl font-bold text-white">T</span>
          </div>
          <span className="text-xl font-bold text-gray-900 dark:text-gray-100">TreeBox</span>
        </Link>

        <div className="flex items-center gap-4">
          {loading ? (
            <div className="h-9 w-24 animate-pulse rounded-md bg-gray-200 dark:bg-gray-800" />
          ) : user ? (
            <>
              <Link href={`/box/${user.domain}`}>
                <Button variant="ghost" size="sm">
                  我的提问箱
                </Button>
              </Link>
              <Link href="/user/questions">
                <Button variant="ghost" size="sm">
                  问题管理
                </Button>
              </Link>
              <Link href="/user/profile">
                <Button variant="ghost" size="sm">
                  设置
                </Button>
              </Link>
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
              <Link href="/login">
                <Button variant="ghost" size="sm">
                  登录
                </Button>
              </Link>
              <Link href="/register">
                <Button size="sm">注册</Button>
              </Link>
            </div>
          )}
        </div>
      </div>
    </nav>
  )
}
