"use client"

import { useEffect, useRef, useState } from "react"
import { useRouter } from "next/navigation"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { useToast } from "@/components/ui/toast"
import { useAuth } from "@/components/auth/auth-provider"
import { api } from "@/lib/api"

const MAX_UPLOAD_SIZE = 10 * 1024 * 1024

type UploadKind = "avatar" | "background"

export default function UserProfilePage() {
  const router = useRouter()
  const { toast } = useToast()
  const { user, setUser, logout, loading: authLoading } = useAuth()

  const avatarInputRef = useRef<HTMLInputElement | null>(null)
  const backgroundInputRef = useRef<HTMLInputElement | null>(null)

  const [savingProfile, setSavingProfile] = useState(false)
  const [savingHarassment, setSavingHarassment] = useState(false)
  const [uploadingAvatar, setUploadingAvatar] = useState(false)
  const [uploadingBackground, setUploadingBackground] = useState(false)
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

  const handleUpdateProfile = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const name = formData.name.trim()
    const intro = formData.intro.trim()

    if (!name) {
      toast("昵称不能为空", "warning")
      return
    }

    if (formData.new_password && !formData.old_password) {
      toast("修改密码前请先输入当前密码", "warning")
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

  const handleUpdateHarassment = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
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
      router.replace("/login")
    } catch (error) {
      const message = error instanceof Error ? error.message : "退出失败，请稍后重试"
      toast(message, "error")
    }
  }

  const handleUpload = async (kind: UploadKind, file?: File | null) => {
    if (!file) {
      return
    }

    if (!file.type.startsWith("image/")) {
      toast("请选择图片文件", "warning")
      return
    }

    if (file.size > MAX_UPLOAD_SIZE) {
      toast("图片不能超过 10MB", "warning")
      return
    }

    if (kind === "avatar") {
      setUploadingAvatar(true)
    } else {
      setUploadingBackground(true)
    }

    try {
      const response =
        kind === "avatar" ? await api.user.uploadAvatar(file) : await api.user.uploadBackground(file)

      if (response.user) {
        setUser(response.user)
      }

      toast(response.message || (kind === "avatar" ? "头像已更新" : "背景已更新"), "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "上传失败，请稍后重试"
      toast(message, "error")
    } finally {
      if (kind === "avatar") {
        setUploadingAvatar(false)
      } else {
        setUploadingBackground(false)
      }
    }
  }

  const handleExportData = async () => {
    setExporting(true)

    try {
      const response = await api.user.exportData()
      const blob = new Blob([JSON.stringify(response, null, 2)], {
        type: "application/json;charset=utf-8",
      })
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
    const confirmed = window.confirm("确定要停用账号吗？这个操作不可撤销。")
    if (!confirmed) {
      return
    }

    setDeactivating(true)

    try {
      const response = await api.user.deactivate()
      if (response.success) {
        await logout()
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
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
          <p className="mt-4 text-gray-600 dark:text-gray-400">正在加载资料...</p>
        </div>
      </div>
    )
  }

  const coverStyle = user.background
    ? {
        backgroundImage: `linear-gradient(135deg, rgba(29, 78, 216, 0.55), rgba(8, 145, 178, 0.4)), url(${user.background})`,
        backgroundPosition: "center",
        backgroundSize: "cover",
      }
    : undefined

  return (
    <section className="container mx-auto px-4 py-8">
      <div className="mx-auto max-w-5xl space-y-6">
        <Card className="overflow-hidden border-0 shadow-xl">
          <div className="h-56 bg-gradient-to-br from-blue-600 via-sky-500 to-cyan-400" style={coverStyle} />
          <CardContent className="relative -mt-16 pb-8">
            <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
                <Avatar className="h-32 w-32 border-4 border-white shadow-2xl">
                  <AvatarImage src={user.avatar} alt={user.name} />
                  <AvatarFallback className="text-4xl">{user.name.charAt(0).toUpperCase()}</AvatarFallback>
                </Avatar>
                <div className="space-y-3">
                  <div>
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-50">{user.name}</h1>
                    <p className="text-sm text-gray-500 dark:text-gray-400">{user.email}</p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button type="button" variant="outline" onClick={() => router.push(`/box/${user.domain}`)}>
                      查看我的提问箱
                    </Button>
                    <Button type="button" variant="ghost" onClick={handleLogout}>
                      退出登录
                    </Button>
                  </div>
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2 lg:justify-end">
                <input
                  ref={avatarInputRef}
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  className="hidden"
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    void handleUpload("avatar", file)
                    event.target.value = ""
                  }}
                />
                <input
                  ref={backgroundInputRef}
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  className="hidden"
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    void handleUpload("background", file)
                    event.target.value = ""
                  }}
                />
                <Button
                  type="button"
                  size="sm"
                  variant="secondary"
                  className="rounded-full bg-white/90 px-4 text-slate-900 shadow-sm hover:bg-white dark:bg-gray-950/80 dark:text-gray-100 dark:hover:bg-gray-950"
                  disabled={uploadingAvatar}
                  onClick={() => avatarInputRef.current?.click()}
                >
                  {uploadingAvatar ? "上传中..." : "上传头像"}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="secondary"
                  className="rounded-full bg-white/90 px-4 text-slate-900 shadow-sm hover:bg-white dark:bg-gray-950/80 dark:text-gray-100 dark:hover:bg-gray-950"
                  disabled={uploadingBackground}
                  onClick={() => backgroundInputRef.current?.click()}
                >
                  {uploadingBackground ? "上传中..." : "上传背景"}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="grid gap-6 lg:grid-cols-[1.3fr,0.9fr]">
          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
              <CardDescription>这里会同步到你的提问箱公开页。</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleUpdateProfile} className="space-y-5">
                <div className="space-y-2">
                  <label className="text-sm font-medium">昵称</label>
                  <Input
                    value={formData.name}
                    maxLength={20}
                    placeholder="给自己起一个容易记住的名字"
                    onChange={(event) => setFormData({ ...formData, name: event.target.value })}
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">提问箱简介</label>
                  <Textarea
                    value={formData.intro}
                    rows={4}
                    maxLength={200}
                    placeholder="告诉大家你希望收到怎样的问题"
                    onChange={(event) => setFormData({ ...formData, intro: event.target.value })}
                  />
                  <p className="text-right text-xs text-gray-500">{formData.intro.length}/200</p>
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">当前密码</label>
                    <Input
                      type="password"
                      value={formData.old_password}
                      placeholder="如需修改密码，请先输入当前密码"
                      onChange={(event) => setFormData({ ...formData, old_password: event.target.value })}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">新密码</label>
                    <Input
                      type="password"
                      value={formData.new_password}
                      placeholder="不修改可以留空"
                      onChange={(event) => setFormData({ ...formData, new_password: event.target.value })}
                    />
                  </div>
                </div>

                <label className="flex items-center gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm dark:border-gray-800 dark:bg-gray-900">
                  <input
                    type="checkbox"
                    checked={formData.notify_email}
                    className="h-4 w-4 rounded border-gray-300"
                    onChange={(event) => setFormData({ ...formData, notify_email: event.target.checked })}
                  />
                  <span>有人提问时，通过邮箱提醒我</span>
                </label>

                <Button type="submit" disabled={savingProfile}>
                  {savingProfile ? "保存中..." : "保存资料"}
                </Button>
              </form>
            </CardContent>
          </Card>

          <div className="space-y-6">
            <Card className="shadow-md">
              <CardHeader>
                <CardTitle>提问限制</CardTitle>
                <CardDescription>你可以限制提问来源，也可以设置敏感词过滤。</CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleUpdateHarassment} className="space-y-4">
                  <label className="flex items-center gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm dark:border-gray-800 dark:bg-gray-900">
                    <input
                      type="checkbox"
                      checked={formData.register_only}
                      className="h-4 w-4 rounded border-gray-300"
                      onChange={(event) => setFormData({ ...formData, register_only: event.target.checked })}
                    />
                    <span>仅允许已登录用户向我提问</span>
                  </label>

                  <div className="space-y-2">
                    <label className="text-sm font-medium">屏蔽词</label>
                    <Input
                      value={formData.block_words}
                      placeholder="多个关键词请用英文逗号分隔"
                      onChange={(event) => setFormData({ ...formData, block_words: event.target.value })}
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      最多 10 个，每个关键词不超过 10 个字符。
                    </p>
                  </div>

                  <Button type="submit" disabled={savingHarassment}>
                    {savingHarassment ? "保存中..." : "保存限制"}
                  </Button>
                </form>
              </CardContent>
            </Card>

            <Card className="shadow-md">
              <CardHeader>
                <CardTitle>账号操作</CardTitle>
                <CardDescription>导出你的数据，或停用当前账号。</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <Button variant="outline" className="w-full" onClick={handleExportData} disabled={exporting}>
                  {exporting ? "导出中..." : "导出我的数据"}
                </Button>

                <Separator />

                <Button variant="outline" className="w-full" onClick={handleDeactivate} disabled={deactivating}>
                  {deactivating ? "处理中..." : "停用账号"}
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </section>
  )
}
