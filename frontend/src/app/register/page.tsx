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

const DOMAIN_PATTERN = /^[a-z0-9](?:[a-z0-9-]{1,18}[a-z0-9])?$/

export default function RegisterPage() {
  const router = useRouter()
  const { toast } = useToast()
  const { setUser, user, loading: authLoading } = useAuth()
  const [loading, setLoading] = useState(false)
  const [formData, setFormData] = useState({
    name: "",
    email: "",
    password: "",
    domain: "",
  })

  useEffect(() => {
    if (!authLoading && user) {
      router.replace("/")
    }
  }, [authLoading, router, user])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const payload = {
      name: formData.name.trim(),
      email: formData.email.trim(),
      password: formData.password,
      domain: formData.domain.trim().toLowerCase(),
      recaptcha: "test",
    }

    if (!payload.name) {
      toast("请输入昵称", "warning")
      return
    }
    if (!payload.email) {
      toast("请输入邮箱地址", "warning")
      return
    }
    if (payload.password.length < 6) {
      toast("密码至少需要 6 位", "warning")
      return
    }
    if (!DOMAIN_PATTERN.test(payload.domain)) {
      toast("个性域名需为 3-20 位小写字母、数字或连字符，且不能以连字符开头或结尾", "warning")
      return
    }

    setLoading(true)

    try {
      const response = await api.auth.register(payload)

      if (!response.user) {
        throw new Error("注册成功，但未获取到用户信息")
      }

      setUser(response.user)
      toast(response.message || "注册成功", "success")
      router.replace("/")
    } catch (error) {
      const message = error instanceof Error ? error.message : "注册失败，请稍后重试"
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
              <span className="text-3xl font-bold text-white">N</span>
            </div>
            <CardTitle className="text-2xl">创建账户</CardTitle>
            <CardDescription>几步完成注册，立即拥有自己的匿名提问箱。</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">昵称</label>
                <Input
                  type="text"
                  placeholder="你的昵称"
                  autoComplete="nickname"
                  maxLength={20}
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>
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
                  placeholder="至少 6 位密码"
                  autoComplete="new-password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">个性域名</label>
                <Input
                  type="text"
                  placeholder="your-name"
                  autoCapitalize="none"
                  autoCorrect="off"
                  value={formData.domain}
                  onChange={(e) => setFormData({ ...formData, domain: e.target.value.toLowerCase() })}
                  required
                />
                <p className="text-xs text-gray-500">
                  这会成为你的提问箱地址，例如 `/box/your-name`。
                </p>
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "注册中..." : "注册"}
              </Button>
            </form>
            <div className="mt-4 text-center text-sm">
              <span className="text-gray-600 dark:text-gray-400">已经有账号了？</span>
              <Link href="/login" className="ml-1 font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400">
                立即登录
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>

      <Footer />
    </div>
  )
}
