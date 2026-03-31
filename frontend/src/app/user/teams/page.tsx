"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { useRouter } from "next/navigation"

import { useAuth } from "@/components/auth/auth-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, type SelectOption } from "@/components/ui/select"
import { useToast } from "@/components/ui/toast"
import { api } from "@/lib/api"
import type { TenantMemberSummary, TenantSummary } from "@/types"

const ROLE_OPTIONS = ["admin", "member", "viewer"] as const

const ROLE_LABELS: Record<string, string> = {
  admin: "管理员",
  member: "成员",
  viewer: "只读成员",
}

export default function UserTeamsPage() {
  const router = useRouter()
  const { toast } = useToast()
  const { user, loading: authLoading } = useAuth()

  const [loading, setLoading] = useState(true)
  const [loadingMembers, setLoadingMembers] = useState(false)
  const [adding, setAdding] = useState(false)
  const [tenants, setTenants] = useState<TenantSummary[]>([])
  const [selectedTenantUID, setSelectedTenantUID] = useState("")
  const [members, setMembers] = useState<TenantMemberSummary[]>([])
  const [memberForm, setMemberForm] = useState({
    email: "",
    role: "member",
  })

  const selectedTenant = useMemo(
    () => tenants.find((tenant) => tenant.uid === selectedTenantUID) || null,
    [selectedTenantUID, tenants]
  )

  const tenantOptions = useMemo<SelectOption[]>(
    () =>
      tenants.map((tenant) => ({
        value: tenant.uid,
        label: tenant.name,
        description: ROLE_LABELS[tenant.role] || tenant.role,
      })),
    [tenants]
  )

  const roleOptions = useMemo<SelectOption[]>(
    () =>
      ROLE_OPTIONS.map((role) => ({
        value: role,
        label: ROLE_LABELS[role] || role,
      })),
    []
  )

  const loadMembers = useCallback(
    async (tenantUID: string) => {
      if (!tenantUID) {
        setMembers([])
        return
      }

      setLoadingMembers(true)

      try {
        const response = await api.tenants.members.list(tenantUID)
        setMembers(response.members)
      } catch (error) {
        const message = error instanceof Error ? error.message : "加载租户成员失败，请稍后重试。"
        toast(message, "error")
      } finally {
        setLoadingMembers(false)
      }
    },
    [toast]
  )

  useEffect(() => {
    if (authLoading) {
      return
    }

    if (!user) {
      router.replace("/login?to=/user/teams")
      return
    }

    let cancelled = false

    const bootstrap = async () => {
      try {
        const response = await api.tenants.list()
        if (cancelled) {
          return
        }

        setTenants(response.tenants)
        const initialTenantUID = response.tenants[0]?.uid || ""
        setSelectedTenantUID(initialTenantUID)
        if (initialTenantUID) {
          await loadMembers(initialTenantUID)
        }
      } catch (error) {
        if (!cancelled) {
          const message = error instanceof Error ? error.message : "加载租户列表失败，请稍后重试。"
          toast(message, "error")
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void bootstrap()

    return () => {
      cancelled = true
    }
  }, [authLoading, loadMembers, router, toast, user])

  const handleTenantChange = async (tenantUID: string) => {
    setSelectedTenantUID(tenantUID)
    await loadMembers(tenantUID)
  }

  const handleAddMember = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!selectedTenantUID) {
      toast("请先选择一个租户。", "warning")
      return
    }

    const email = memberForm.email.trim()
    if (!email) {
      toast("成员邮箱不能为空。", "warning")
      return
    }

    setAdding(true)

    try {
      const response = await api.tenants.members.add(selectedTenantUID, {
        email,
        role: memberForm.role,
      })
      setMembers((current) => [...current, response.member])
      setMemberForm({
        email: "",
        role: "member",
      })
      toast(response.message || "成员添加成功。", "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "添加成员失败，请稍后重试。"
      toast(message, "error")
    } finally {
      setAdding(false)
    }
  }

  const handleUpdateRole = async (member: TenantMemberSummary, role: string) => {
    if (!selectedTenantUID || role === member.role) {
      return
    }

    try {
      const response = await api.tenants.members.updateRole(selectedTenantUID, member.user_id, { role })
      setMembers((current) =>
        current.map((currentMember) => (currentMember.user_id === member.user_id ? response.member : currentMember))
      )
      toast(response.message || "成员角色已更新。", "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "更新成员角色失败，请稍后重试。"
      toast(message, "error")
    }
  }

  const handleRemoveMember = async (member: TenantMemberSummary) => {
    if (!selectedTenantUID) {
      return
    }

    const confirmed = window.confirm(`确认将 ${member.email} 从 ${selectedTenant?.name || "当前租户"} 中移除吗？`)
    if (!confirmed) {
      return
    }

    try {
      const response = await api.tenants.members.remove(selectedTenantUID, member.user_id)
      setMembers((current) => current.filter((currentMember) => currentMember.user_id !== member.user_id))
      toast(response.message || "成员已移除。", "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "移除成员失败，请稍后重试。"
      toast(message, "error")
    }
  }

  if (authLoading || loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-cyan-500 border-t-transparent" />
          <p className="mt-4 text-slate-600 dark:text-slate-400">正在加载租户成员...</p>
        </div>
      </div>
    )
  }

  if (!user) {
    return null
  }

  return (
    <section className="container mx-auto px-4 py-8">
      <div className="mx-auto max-w-6xl space-y-6">
        <Card className="overflow-hidden border-0 shadow-xl">
          <div className="bg-[radial-gradient(circle_at_top_right,_rgba(20,184,166,0.22),_transparent_42%),linear-gradient(135deg,#111827,#155e75)] px-6 py-8 text-white">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div className="space-y-3">
                <Badge className="bg-white/15 text-white hover:bg-white/15">租户协作</Badge>
                <div>
                  <h1 className="text-3xl font-semibold tracking-tight">团队成员管理</h1>
                  <p className="mt-2 max-w-2xl text-sm text-cyan-50/80">
                    邀请协作者加入租户，统一管理权限边界，让多租户团队模型真正支撑客服、运营、销售等协作流程。
                  </p>
                </div>
              </div>

              <div className="rounded-3xl border border-white/10 bg-white/10 px-5 py-4 backdrop-blur">
                <p className="text-xs uppercase tracking-[0.28em] text-cyan-100/70">可管理租户</p>
                <p className="mt-2 text-3xl font-semibold">{tenants.length}</p>
              </div>
            </div>
          </div>
        </Card>

        <div className="grid gap-6 lg:grid-cols-[1.35fr,0.95fr]">
          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>成员列表</CardTitle>
              <CardDescription>
                所有者和管理员可以管理成员；成员可以参与问题处理；只读成员仅可查看数据，不参与处理。
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700 dark:text-slate-200">选择租户</label>
                <Select
                  value={selectedTenantUID}
                  onChange={(tenantUID) => void handleTenantChange(tenantUID)}
                  options={tenantOptions}
                  placeholder="请选择租户"
                />
              </div>

              {loadingMembers ? (
                <div className="rounded-3xl border border-dashed border-slate-300 px-6 py-12 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                  正在加载成员列表...
                </div>
              ) : members.length === 0 ? (
                <div className="rounded-3xl border border-dashed border-slate-300 px-6 py-12 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                  当前租户下还没有成员数据。
                </div>
              ) : (
                members.map((member) => (
                  <div
                    key={`${member.user_id}-${member.email}`}
                    className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-950"
                  >
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                      <div className="space-y-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <h2 className="text-base font-semibold text-slate-950 dark:text-white">{member.name}</h2>
                          <Badge variant="outline">{ROLE_LABELS[member.role] || member.role}</Badge>
                        </div>
                        <p className="text-sm text-slate-500 dark:text-slate-400">{member.email}</p>
                        <p className="text-xs text-slate-500 dark:text-slate-400">@{member.domain}</p>
                      </div>

                      <div className="flex flex-wrap items-center gap-2">
                        <Select
                          value={member.role}
                          onChange={(role) => void handleUpdateRole(member, role)}
                          options={roleOptions}
                          size="sm"
                          className="min-w-[9rem]"
                        />
                        <Button type="button" variant="outline" onClick={() => void handleRemoveMember(member)}>
                          移除
                        </Button>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </CardContent>
          </Card>

          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>邀请成员</CardTitle>
              <CardDescription>
                将已注册用户加入当前租户，并决定他是参与处理、协作支持，还是仅保留只读权限。
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form className="space-y-4" onSubmit={handleAddMember}>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700 dark:text-slate-200">成员邮箱</label>
                  <Input
                    value={memberForm.email}
                    placeholder="请输入已注册用户的邮箱"
                    onChange={(event) => setMemberForm((current) => ({ ...current, email: event.target.value }))}
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700 dark:text-slate-200">角色</label>
                  <Select
                    value={memberForm.role}
                    onChange={(role) => setMemberForm((current) => ({ ...current, role }))}
                    options={roleOptions}
                  />
                </div>

                <Button type="submit" className="w-full" disabled={adding}>
                  {adding ? "添加中..." : "添加成员"}
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </section>
  )
}
