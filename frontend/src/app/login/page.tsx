"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Header } from "@/components/layout/header"
import { Footer } from "@/components/layout/footer"
import { useToast } from "@/components/ui/toast"
import { useAuth } from "@/components/auth/auth-provider"
import { api } from "@/lib/api"

export default function LoginPage() {
  const router = useRouter()
  const { toast } = useToast()
  const { setUser, user, loading: authLoading } = useAuth()
  const [loading, setLoading] = useState(false)
  const [redirectTo, setRedirectTo] = useState<string | null>(null)
  const [formData, setFormData] = useState({
    email: "",
    password: "",
  })

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    setRedirectTo(params.get("to")?.trim() || null)
  }, [])

  useEffect(() => {
    if (!authLoading && user) {
      router.replace(redirectTo || `/box/${user.domain}`)
    }
  }, [authLoading, redirectTo, router, user])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const email = formData.email.trim()
    const password = formData.password

    if (!email) {
      toast("请输入邮箱地址", "warning")
      return
    }
    if (!password) {
      toast("请输入密码", "warning")
      return
    }

    setLoading(true)

    try {
      const response = await api.auth.login({
        email,
        password,
        recaptcha: "test",
      })

      if (!response.user) {
        throw new Error("登录成功，但未获取到用户信息")
      }

      setUser(response.user)
      toast(response.message || "登录成功", "success")
      router.replace(redirectTo || `/box/${response.user.domain}`)
    } catch (error) {
      const message = error instanceof Error ? error.message : "登录失败，请稍后重试"
      toast(message, "error")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <Header />

      <div className="container mx-auto flex min-h-[calc(100vh-16rem)] items-center justify-center px-4 py-16">
        <Card className="w-full max-w-md shadow-xl">
          <CardHeader className="space-y-1 text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-blue-600 to-cyan-500">
              <span className="text-3xl font-bold text-white">T</span>
            </div>
            <CardTitle className="text-2xl">欢迎回来</CardTitle>
            <CardDescription>登录到你的 TreeBox 账户，继续管理提问箱。</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">邮箱</label>
                <Input
                  type="email"
                  placeholder="your@email.com"
                  autoComplete="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">密码</label>
                <Input
                  type="password"
                  placeholder="请输入密码"
                  autoComplete="current-password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "登录中..." : "登录"}
              </Button>
            </form>
            <div className="mt-4 text-center text-sm">
              <span className="text-gray-600 dark:text-gray-400">还没有账号？</span>
              <Link href="/register" className="ml-1 font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400">
                立即注册
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>

      <Footer />
    </div>
  )
}
