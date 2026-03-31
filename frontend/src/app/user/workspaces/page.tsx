"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { useRouter } from "next/navigation"

import { useAuth } from "@/components/auth/auth-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, type SelectOption } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { useToast } from "@/components/ui/toast"
import { api } from "@/lib/api"
import type { TenantSummary, WorkspaceSummary } from "@/types"

const TENANT_ROLE_LABELS: Record<string, string> = {
  owner: "所有者",
  admin: "管理员",
  member: "成员",
  viewer: "只读成员",
}

export default function UserWorkspacesPage() {
  const router = useRouter()
  const { toast } = useToast()
  const { user, setUser, loading: authLoading } = useAuth()

  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [activatingUID, setActivatingUID] = useState("")
  const [tenants, setTenants] = useState<TenantSummary[]>([])
  const [workspaces, setWorkspaces] = useState<WorkspaceSummary[]>([])
  const [formData, setFormData] = useState({
    tenant_uid: "",
    name: "",
    description: "",
  })

  const loadData = useCallback(async () => {
    const [tenantResponse, workspaceResponse] = await Promise.all([api.tenants.list(), api.workspaces.list()])
    setTenants(tenantResponse.tenants)
    setWorkspaces(workspaceResponse.workspaces)

    setFormData((current) => ({
      ...current,
      tenant_uid: current.tenant_uid || tenantResponse.tenants[0]?.uid || "",
    }))
  }, [])

  useEffect(() => {
    if (authLoading) {
      return
    }

    if (!user) {
      router.replace("/login?to=/user/workspaces")
      return
    }

    let cancelled = false

    const bootstrap = async () => {
      try {
        await loadData()
      } catch (error) {
        if (!cancelled) {
          const message = error instanceof Error ? error.message : "加载工作区失败，请稍后重试。"
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
  }, [authLoading, loadData, router, toast, user])

  const tenantStats = useMemo(() => {
    const uniqueTenantIDs = new Set(workspaces.map((workspace) => workspace.tenant.uid))
    return {
      workspaceCount: workspaces.length,
      tenantCount: uniqueTenantIDs.size,
    }
  }, [workspaces])

  const tenantOptions = useMemo<SelectOption[]>(
    () =>
      tenants.map((tenant) => ({
        value: tenant.uid,
        label: tenant.name,
        description: TENANT_ROLE_LABELS[tenant.role] || tenant.role,
      })),
    [tenants]
  )

  const handleCreateWorkspace = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const name = formData.name.trim()
    if (!formData.tenant_uid) {
      toast("请先选择一个租户。", "warning")
      return
    }
    if (!name) {
      toast("工作区名称不能为空。", "warning")
      return
    }

    setCreating(true)

    try {
      const response = await api.workspaces.create({
        tenant_uid: formData.tenant_uid,
        name,
        description: formData.description.trim(),
      })

      setWorkspaces((current) => [...current, response.workspace])
      setFormData((current) => ({
        ...current,
        name: "",
        description: "",
      }))
      toast(response.message || "工作区创建成功。", "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "创建工作区失败，请稍后重试。"
      toast(message, "error")
    } finally {
      setCreating(false)
    }
  }

  const handleSetIntake = async (workspace: WorkspaceSummary) => {
    setActivatingUID(workspace.uid)

    try {
      const response = await api.workspaces.setIntake(workspace.uid)
      if (response.user) {
        setUser(response.user)
      }
      toast(response.message || `已将 ${workspace.name} 设为新问题接收入口。`, "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : "设置接收工作区失败，请稍后重试。"
      toast(message, "error")
    } finally {
      setActivatingUID("")
    }
  }

  if (authLoading || loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-cyan-500 border-t-transparent" />
          <p className="mt-4 text-slate-600 dark:text-slate-400">正在加载工作区控制台...</p>
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
          <div className="bg-[radial-gradient(circle_at_top_left,_rgba(6,182,212,0.18),_transparent_45%),linear-gradient(135deg,#0f172a,#0f766e)] px-6 py-8 text-white">
            <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
              <div className="space-y-3">
                <Badge className="bg-white/15 text-white hover:bg-white/15">SaaS 控制台</Badge>
                <div>
                  <h1 className="text-3xl font-semibold tracking-tight">工作区运营台</h1>
                  <p className="mt-2 max-w-2xl text-sm text-cyan-50/80">
                    将不同业务线的问题路由到独立工作区，按租户维度统一管理客服、销售、运营等团队的收件箱与处理流程。
                  </p>
                </div>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-3xl border border-white/10 bg-white/10 px-4 py-3 backdrop-blur">
                  <p className="text-xs uppercase tracking-[0.28em] text-cyan-100/70">租户数</p>
                  <p className="mt-2 text-3xl font-semibold">{tenantStats.tenantCount}</p>
                </div>
                <div className="rounded-3xl border border-white/10 bg-white/10 px-4 py-3 backdrop-blur">
                  <p className="text-xs uppercase tracking-[0.28em] text-cyan-100/70">工作区数</p>
                  <p className="mt-2 text-3xl font-semibold">{tenantStats.workspaceCount}</p>
                </div>
              </div>
            </div>
          </div>
        </Card>

        <div className="grid gap-6 lg:grid-cols-[1.35fr,0.95fr]">
          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>工作区列表</CardTitle>
              <CardDescription>
                每个工作区都可以看作一条独立的业务处理链路。你可以进入团队收件箱，也可以把它设为新问题的默认接收入口。
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {workspaces.length === 0 ? (
                <div className="rounded-3xl border border-dashed border-slate-300 px-6 py-12 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                  还没有工作区，可以先在右侧创建一个。
                </div>
              ) : (
                workspaces.map((workspace) => {
                  const isIntake = user.routing_workspace_id === workspace.id

                  return (
                    <div
                      key={workspace.uid}
                      className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-cyan-200 hover:shadow-md dark:border-slate-800 dark:bg-slate-950"
                    >
                      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                        <div className="space-y-3">
                          <div className="flex flex-wrap items-center gap-2">
                            <h2 className="text-lg font-semibold text-slate-950 dark:text-white">{workspace.name}</h2>
                            {workspace.is_default ? <Badge variant="secondary">默认工作区</Badge> : null}
                            {isIntake ? <Badge>当前接收入口</Badge> : null}
                            <Badge variant="outline">{TENANT_ROLE_LABELS[workspace.tenant.role] || workspace.tenant.role}</Badge>
                          </div>
                          <p className="text-sm text-slate-500 dark:text-slate-400">{workspace.tenant.name}</p>
                          <p className="text-sm leading-6 text-slate-600 dark:text-slate-300">
                            {workspace.description?.trim() || "暂未填写工作区说明。"}
                          </p>
                        </div>

                        <div className="flex flex-wrap gap-2">
                          <Button type="button" variant="outline" onClick={() => router.push(`/user/workspaces/${workspace.uid}`)}>
                            进入收件箱
                          </Button>
                          <Button
                            type="button"
                            disabled={isIntake || activatingUID === workspace.uid}
                            onClick={() => void handleSetIntake(workspace)}
                          >
                            {activatingUID === workspace.uid ? "切换中..." : isIntake ? "正在接收" : "设为接收入口"}
                          </Button>
                        </div>
                      </div>
                    </div>
                  )
                })
              )}
            </CardContent>
          </Card>

          <Card className="shadow-md">
            <CardHeader>
              <CardTitle>创建工作区</CardTitle>
              <CardDescription>
                成熟的 SaaS 平台通常会把客服、销售、成功团队、内容审核等场景拆分到不同工作区中，形成清晰的运营边界。
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form className="space-y-4" onSubmit={handleCreateWorkspace}>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700 dark:text-slate-200">所属租户</label>
                  <Select
                    value={formData.tenant_uid}
                    onChange={(tenantUID) => setFormData((current) => ({ ...current, tenant_uid: tenantUID }))}
                    options={tenantOptions}
                    placeholder="请选择租户"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700 dark:text-slate-200">工作区名称</label>
                  <Input
                    value={formData.name}
                    placeholder="例如：客服运营 / 销售线索 / 社区审核"
                    onChange={(event) => setFormData((current) => ({ ...current, name: event.target.value }))}
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700 dark:text-slate-200">工作区说明</label>
                  <Textarea
                    rows={5}
                    value={formData.description}
                    placeholder="例如：负责处理产品反馈、需求收集、问题分派与统一答复。"
                    onChange={(event) => setFormData((current) => ({ ...current, description: event.target.value }))}
                  />
                </div>

                <Button type="submit" className="w-full" disabled={creating}>
                  {creating ? "创建中..." : "创建工作区"}
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </section>
  )
}
