"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Separator } from "@/components/ui/separator"
import { useToast } from "@/components/ui/toast"
import { useAuth } from "@/components/auth/auth-provider"
import { api } from "@/lib/api"

export default function UserProfilePage() {
  const router = useRouter()
  const { toast } = useToast()
  const { user, setUser, logout, loading: authLoading } = useAuth()

  const [savingProfile, setSavingProfile] = useState(false)
  const [savingHarassment, setSavingHarassment] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [deactivating, setDeactivating] = useState(false)
  const [formData, setFormData] = useState({
    name: "",
    intro: "",
    old_password: "",
    new_password: "",
    notify_email: false,
    register_only: false,
    block_words: "",
  })

  useEffect(() => {
    if (authLoading) {
      return
    }

    if (!user) {
      router.replace("/login?to=/user/profile")
      return
    }

    setFormData({
      name: user.name,
      intro: user.intro,
      old_password: "",
      new_password: "",
      notify_email: user.notify === "email",
      register_only: user.harassment_setting === "register_only",
      block_words: user.block_words || "",
    })
  }, [authLoading, router, user])

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault()

    const name = formData.name.trim()
    const intro = formData.intro.trim()

    if (!name) {
      toast("昵称不能为空", "warning")
      return
    }
    if (formData.new_password && !formData.old_password) {
      toast("修改密码前请输入当前密码", "warning")
      return
    }

    setSavingProfile(true)

    try {
      const response = await api.user.updateProfile({
        name,
        intro,
        old_password: formData.old_password,
        new_password: formData.new_password,
        notify_email: formData.notify_email,
      })

      if (response.user) {
        setUser(response.user)
      }
      toast(response.message || "资料已更新", "success")
      setFormData((current) => ({
        ...current,
        name,
        intro,
        old_password: "",
        new_password: "",
      }))
    } catch (error) {
      const message = error instanceof Error ? error.message : "更新失败，请稍后重试"
      toast(message, "error")
    } finally {
      setSavingProfile(false)
    }
  }

  const handleUpdateHarassment = async (e: React.FormEvent) => {
    e.preventDefault()
    setSavingHarassment(true)

    try {
      const response = await api.user.updateHarassment({
        register_only: formData.register_only,
        block_words: formData.block_words,
      })

      if (response.user) {
        setUser(response.user)
      }
      toast(response.message || "提问限制已更新", "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "更新失败，请稍后重试"
      toast(message, "error")
    } finally {
      setSavingHarassment(false)
    }
  }

  const handleLogout = async () => {
    try {
      await logout()
      toast("已退出登录", "success")
      router.replace("/login")
    } catch (error) {
      const message = error instanceof Error ? error.message : "退出失败，请稍后重试"
      toast(message, "error")
    }
  }

  const handleExportData = async () => {
    setExporting(true)
    try {
      const res = await api.user.exportData()
      const blob = new Blob([JSON.stringify(res, null, 2)], { type: "application/json;charset=utf-8" })
      const url = URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = `treebox-export-${new Date().toISOString().slice(0, 10)}.json`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
      toast("数据导出成功", "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "导出失败，请稍后重试"
      toast(message, "error")
    } finally {
      setExporting(false)
    }
  }

  const handleDeactivate = async () => {
    const confirmed = window.confirm("确定要停用账号吗？停用后将无法登录，且该操作不可撤销。")
    if (!confirmed) {
      return
    }

    setDeactivating(true)
    try {
      const response = await api.user.deactivate()
      if (response.success) {
        await logout()
        toast("账号已停用", "success")
        router.replace("/login")
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "停用失败，请稍后重试"
      toast(message, "error")
    } finally {
      setDeactivating(false)
    }
  }

  if (authLoading || !user) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
        <div className="flex min-h-[60vh] items-center justify-center">
          <div className="text-center">
            <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
            <p className="mt-4 text-gray-600 dark:text-gray-400">加载中...</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <div className="container mx-auto px-4 py-8">
        <div className="mx-auto max-w-2xl space-y-6">
          <Card className="shadow-lg">
            <CardHeader>
              <div className="flex items-center gap-4">
                <Avatar className="h-16 w-16">
                  <AvatarImage src={user.avatar || "/default-avatar.png"} alt={user.name} />
                  <AvatarFallback className="text-2xl">{user.name.charAt(0).toUpperCase()}</AvatarFallback>
                </Avatar>
                <div>
                  <CardTitle>个人设置</CardTitle>
                  <CardDescription>{user.email}</CardDescription>
                </div>
              </div>
            </CardHeader>
          </Card>

          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleUpdateProfile} className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">昵称</label>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="你的昵称"
                    maxLength={20}
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">提问箱介绍</label>
                  <Input
                    value={formData.intro}
                    onChange={(e) => setFormData({ ...formData, intro: e.target.value })}
                    placeholder="简单介绍一下你的提问箱"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">修改密码</label>
                  <div className="space-y-2">
                    <Input
                      type="password"
                      value={formData.old_password}
                      onChange={(e) => setFormData({ ...formData, old_password: e.target.value })}
                      placeholder="当前密码，不修改可留空"
                    />
                    <Input
                      type="password"
                      value={formData.new_password}
                      onChange={(e) => setFormData({ ...formData, new_password: e.target.value })}
                      placeholder="新密码，不修改可留空"
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">新提问通知</label>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={formData.notify_email}
                      onChange={(e) => setFormData({ ...formData, notify_email: e.target.checked })}
                      className="h-4 w-4 rounded border-gray-300"
                    />
                    <span className="text-sm">通过邮件接收新提问通知</span>
                  </label>
                </div>

                <div className="flex gap-2">
                  <Button type="submit" disabled={savingProfile}>
                    {savingProfile ? "保存中..." : "保存修改"}
                  </Button>
                  <Button type="button" variant="outline" onClick={handleLogout}>
                    退出登录
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>

          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>提问限制</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleUpdateHarassment} className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">谁可以向我提问</label>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={formData.register_only}
                      onChange={(e) => setFormData({ ...formData, register_only: e.target.checked })}
                      className="h-4 w-4 rounded border-gray-300"
                    />
                    <span className="text-sm">仅允许已注册用户向我提问</span>
                  </label>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">屏蔽词</label>
                  <Input
                    value={formData.block_words}
                    onChange={(e) => setFormData({ ...formData, block_words: e.target.value })}
                    placeholder="使用英文逗号分隔，最多 10 个，每个最多 10 个字"
                  />
                  <p className="text-xs text-gray-500">
                    命中屏蔽词的提问会被直接拦截，适合过滤广告、骚扰内容或不希望出现的关键词。
                  </p>
                </div>

                <Button type="submit" disabled={savingHarassment}>
                  {savingHarassment ? "保存中..." : "保存设置"}
                </Button>
              </form>
            </CardContent>
          </Card>

          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>账号操作</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Button variant="outline" className="w-full" onClick={handleExportData} disabled={exporting}>
                  {exporting ? "导出中..." : "导出我的全部数据"}
                </Button>
                <p className="mt-2 text-xs text-gray-500">
                  导出的 JSON 文件会包含你的账户资料、收到的问题和已有回答。
                </p>
              </div>

              <Separator />

              <div>
                <Button
                  variant="destructive"
                  className="w-full"
                  onClick={handleDeactivate}
                  disabled={deactivating}
                >
                  {deactivating ? "停用中..." : "停用我的账号"}
                </Button>
                <p className="mt-2 text-xs text-gray-500">
                  停用后你的账号将无法再登录，提问箱页面也会失效。该操作不可恢复，请谨慎操作。
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
