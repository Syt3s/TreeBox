"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { useParams, useRouter } from "next/navigation"

import { useAuth } from "@/components/auth/auth-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, type SelectOption } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { useToast } from "@/components/ui/toast"
import { api } from "@/lib/api"
import type {
  Question,
  TenantMemberSummary,
  WorkspaceQuestionStats,
  WorkspaceSummary,
} from "@/types"

const STATUS_OPTIONS: SelectOption[] = [
  { value: "new", label: "新建" },
  { value: "in_progress", label: "处理中" },
  { value: "answered", label: "已回复" },
  { value: "closed", label: "已关闭" },
]

const TENANT_ROLE_LABELS: Record<string, string> = {
  owner: "所有者",
  admin: "管理员",
  member: "成员",
  viewer: "只读成员",
}

type DraftState = {
  answer: string
  internal_note: string
}

export default function WorkspaceInboxPage() {
  const params = useParams<{ workspaceUID: string }>()
  const router = useRouter()
  const { toast } = useToast()
  const { user, loading: authLoading } = useAuth()

  const workspaceUID = Array.isArray(params?.workspaceUID) ? params.workspaceUID[0] : params?.workspaceUID || ""

  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [workspace, setWorkspace] = useState<WorkspaceSummary | null>(null)
  const [stats, setStats] = useState<WorkspaceQuestionStats | null>(null)
  const [questions, setQuestions] = useState<Question[]>([])
  const [members, setMembers] = useState<TenantMemberSummary[]>([])
  const [drafts, setDrafts] = useState<Record<number, DraftState>>({})
  const [statusFilter, setStatusFilter] = useState("all")
  const [assigneeFilter, setAssigneeFilter] = useState("all")
  const [showPrivate, setShowPrivate] = useState(true)
  const [busyKey, setBusyKey] = useState("")

  const memberByID = useMemo(() => {
    return members.reduce<Record<number, TenantMemberSummary>>((accumulator, member) => {
      accumulator[member.user_id] = member
      return accumulator
    }, {})
  }, [members])

  const statusFilterOptions = useMemo<SelectOption[]>(
    () => [{ value: "all", label: "全部状态" }, ...STATUS_OPTIONS],
    []
  )

  const assigneeFilterOptions = useMemo<SelectOption[]>(
    () => [
      { value: "all", label: "全部成员" },
      { value: "me", label: "只看分配给我" },
      { value: "unassigned", label: "只看未分配" },
      ...members.map((member) => ({
        value: String(member.user_id),
        label: member.name,
        description: TENANT_ROLE_LABELS[member.role] || member.role,
      })),
    ],
    [members]
  )

  const questionAssigneeOptions = useMemo<SelectOption[]>(
    () => [
      { value: "", label: "未分配" },
      ...members.map((member) => ({
        value: String(member.user_id),
        label: member.name,
        description: TENANT_ROLE_LABELS[member.role] || member.role,
      })),
    ],
    [members]
  )

  const syncDrafts = useCallback((items: Question[]) => {
    setDrafts((current) => {
      const next: Record<number, DraftState> = {}
      for (const question of items) {
        next[question.id] = {
          answer: current[question.id]?.answer ?? question.answer ?? "",
          internal_note: current[question.id]?.internal_note ?? question.internal_note ?? "",
        }
      }
      return next
    })
  }, [])

  const replaceQuestion = useCallback((nextQuestion?: Question) => {
    if (!nextQuestion) {
      return
    }

    setQuestions((current) =>
      current.map((question) => (question.id === nextQuestion.id ? { ...question, ...nextQuestion } : question))
    )
    setDrafts((current) => ({
      ...current,
      [nextQuestion.id]: {
        answer: nextQuestion.answer ?? current[nextQuestion.id]?.answer ?? "",
        internal_note: nextQuestion.internal_note ?? current[nextQuestion.id]?.internal_note ?? "",
      },
    }))
  }, [])

  const loadWorkspace = useCallback(
    async (showRefreshState = false) => {
      if (!workspaceUID) {
        return
      }

      if (showRefreshState) {
        setRefreshing(true)
      }

      try {
        const questionParams: Parameters<typeof api.workspaces.questions.list>[1] = {
          page_size: 50,
          show_private: showPrivate,
        }

        if (statusFilter !== "all") {
          questionParams.status = statusFilter
        }

        if (assigneeFilter === "me" && user) {
          questionParams.assigned_to_user_id = user.id
        } else if (assigneeFilter === "unassigned") {
          questionParams.only_unassigned = true
        } else if (assigneeFilter !== "all") {
          const parsedID = Number(assigneeFilter)
          if (!Number.isNaN(parsedID)) {
            questionParams.assigned_to_user_id = parsedID
          }
        }

        const [questionsResponse, statsResponse] = await Promise.all([
          api.workspaces.questions.list(workspaceUID, questionParams),
          api.workspaces.stats(workspaceUID),
        ])

        setWorkspace(questionsResponse.workspace)
        setQuestions(questionsResponse.questions)
        setStats(statsResponse.stats)
        syncDrafts(questionsResponse.questions)

        const membersResponse = await api.tenants.members.list(questionsResponse.workspace.tenant.uid)
        setMembers(membersResponse.members)
      } catch (error) {
        const message = error instanceof Error ? error.message : "加载工作区收件箱失败，请稍后重试。"
        toast(message, "error")
      } finally {
        setLoading(false)
        setRefreshing(false)
      }
    },
    [assigneeFilter, showPrivate, statusFilter, syncDrafts, toast, user, workspaceUID]
  )

  useEffect(() => {
    if (authLoading) {
      return
    }

    if (!user) {
      router.replace(`/login?to=/user/workspaces/${workspaceUID}`)
      return
    }

    void loadWorkspace()
  }, [authLoading, loadWorkspace, router, user, workspaceUID])

  useEffect(() => {
    if (!user || authLoading || loading) {
      return
    }

    void loadWorkspace(true)
  }, [assigneeFilter, authLoading, loadWorkspace, loading, showPrivate, statusFilter, user])

  const handleDraftChange = (questionID: number, field: keyof DraftState, value: string) => {
    setDrafts((current) => ({
      ...current,
      [questionID]: {
        answer: current[questionID]?.answer ?? "",
        internal_note: current[questionID]?.internal_note ?? "",
        [field]: value,
      },
    }))
  }

  const runMutation = async (
    key: string,
    task: () => Promise<{ message?: string; question?: Question }>,
    successFallback: string,
    errorFallback: string
  ) => {
    setBusyKey(key)
    try {
      const response = await task()
      replaceQuestion(response.question)
      await loadWorkspace(true)
      toast(response.message || successFallback, "success")
    } catch (error) {
      const message = error instanceof Error ? error.message : errorFallback
      toast(message, "error")
    } finally {
      setBusyKey("")
    }
  }

  const formatTime = (value?: string | null) => {
    if (!value) {
      return "未设置"
    }

    return new Date(value).toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  if (authLoading || loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-cyan-500 border-t-transparent" />
          <p className="mt-4 text-slate-600 dark:text-slate-400">正在加载工作区收件箱...</p>
        </div>
      </div>
    )
  }

  if (!user || !workspace) {
    return null
  }

  return (
    <section className="container mx-auto px-4 py-8">
      <div className="mx-auto max-w-7xl space-y-6">
        <Card className="overflow-hidden border-0 shadow-xl">
          <div className="bg-[radial-gradient(circle_at_top_right,_rgba(8,145,178,0.28),_transparent_40%),linear-gradient(135deg,#0f172a,#164e63)] px-6 py-8 text-white">
            <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
              <div className="space-y-3">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge className="bg-white/15 text-white hover:bg-white/15">{workspace.tenant.name}</Badge>
                  <Badge variant="outline" className="border-white/25 text-white">
                    {TENANT_ROLE_LABELS[workspace.tenant.role] || workspace.tenant.role}
                  </Badge>
                </div>
                <div>
                  <h1 className="text-3xl font-semibold tracking-tight">{workspace.name}</h1>
                  <p className="mt-2 max-w-3xl text-sm text-cyan-50/80">
                    {workspace.description?.trim() || "这个工作区会作为团队共享的处理队列，统一承接和协作处理问题。"}
                  </p>
                </div>
              </div>

              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" onClick={() => router.push("/user/workspaces")}>
                  返回工作区列表
                </Button>
                <Button
                  variant="outline"
                  className="border-white/20 bg-transparent text-white hover:bg-white/10"
                  onClick={() => void loadWorkspace(true)}
                >
                  {refreshing ? "刷新中..." : "刷新"}
                </Button>
              </div>
            </div>
          </div>
        </Card>

        {stats ? (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
            <StatCard label="总问题数" value={stats.total_count} tone="slate" />
            <StatCard label="新建问题" value={stats.new_count} tone="cyan" />
            <StatCard label="处理中" value={stats.in_progress_count} tone="amber" />
            <StatCard label="已回复" value={stats.answered_count} tone="emerald" />
            <StatCard label="已指派" value={stats.assigned_count} tone="violet" />
          </div>
        ) : null}

        <Card className="shadow-md">
          <CardHeader>
            <CardTitle>队列筛选</CardTitle>
            <CardDescription>通过状态、负责人和隐私范围筛选，让团队更聚焦地处理高优先级问题。</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-[1fr,1fr,auto]">
            <div className="space-y-2">
              <label className="text-sm font-medium text-slate-700 dark:text-slate-200">问题状态</label>
              <Select value={statusFilter} onChange={setStatusFilter} options={statusFilterOptions} />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-slate-700 dark:text-slate-200">处理人</label>
              <Select value={assigneeFilter} onChange={setAssigneeFilter} options={assigneeFilterOptions} />
            </div>

            <label className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200">
              <input
                type="checkbox"
                checked={showPrivate}
                onChange={(event) => setShowPrivate(event.target.checked)}
                className="h-4 w-4 rounded border-slate-300"
              />
              包含私密问题
            </label>
          </CardContent>
        </Card>

        {questions.length === 0 ? (
          <Card className="shadow-md">
            <CardContent className="py-16 text-center">
              <p className="text-lg font-medium text-slate-950 dark:text-white">当前筛选条件下没有匹配的问题。</p>
              <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
                你可以调整筛选条件，或者先回到工作区列表设置问题路由入口。
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {questions.map((question) => {
              const assignedMember = question.assigned_to_user_id ? memberByID[question.assigned_to_user_id] : null
              const draft = drafts[question.id] || {
                answer: question.answer || "",
                internal_note: question.internal_note || "",
              }

              return (
                <Card key={question.id} className="shadow-md">
                  <CardContent className="space-y-6 pt-6">
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                      <div className="space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                          <Badge variant="secondary">#{question.id}</Badge>
                          <Badge variant="outline">
                            {STATUS_OPTIONS.find((option) => option.value === question.status)?.label || question.status}
                          </Badge>
                          {question.is_private ? <Badge variant="outline">私密</Badge> : null}
                          {assignedMember ? <Badge>{assignedMember.name}</Badge> : <Badge variant="outline">未分配</Badge>}
                        </div>
                        <p className="text-base leading-7 text-slate-900 dark:text-slate-100">{question.content}</p>
                        <div className="flex flex-wrap gap-4 text-xs text-slate-500 dark:text-slate-400">
                          <span>创建时间：{formatTime(question.created_at)}</span>
                          <span>解决时间：{formatTime(question.resolved_at)}</span>
                        </div>
                      </div>

                      <div className="grid gap-2 sm:grid-cols-2">
                        <Select
                          value={question.status}
                          onChange={(status) =>
                            void runMutation(
                              `status-${question.id}`,
                              () =>
                                api.workspaces.questions.updateStatus(workspace.uid, question.id, {
                                  status,
                                }),
                              "问题状态已更新。",
                              "更新问题状态失败，请稍后重试。"
                            )
                          }
                          options={STATUS_OPTIONS}
                          size="sm"
                          className="min-w-[9.5rem]"
                          disabled={busyKey === `status-${question.id}`}
                        />

                        <Select
                          value={question.assigned_to_user_id ? String(question.assigned_to_user_id) : ""}
                          onChange={(assigneeValue) => {
                            const assignedToUserID = assigneeValue ? Number(assigneeValue) : null
                            void runMutation(
                              `assignee-${question.id}`,
                              () =>
                                api.workspaces.questions.updateAssignee(workspace.uid, question.id, {
                                  assigned_to_user_id: assignedToUserID,
                                }),
                              "负责人已更新。",
                              "更新负责人失败，请稍后重试。"
                            )
                          }}
                          options={questionAssigneeOptions}
                          size="sm"
                          className="min-w-[11rem]"
                          disabled={busyKey === `assignee-${question.id}`}
                        />
                      </div>
                    </div>

                    <div className="grid gap-6 xl:grid-cols-2">
                      <div className="space-y-3">
                        <div className="flex items-center justify-between">
                          <h2 className="text-sm font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">
                            回复内容
                          </h2>
                          <Button
                            type="button"
                            size="sm"
                            onClick={() =>
                              void runMutation(
                                `answer-${question.id}`,
                                () =>
                                  api.workspaces.questions.answer(workspace.uid, question.id, {
                                    answer: drafts[question.id]?.answer || "",
                                  }),
                                "回复已发布。",
                                "保存回复失败，请稍后重试。"
                              )
                            }
                            disabled={busyKey === `answer-${question.id}`}
                          >
                            {busyKey === `answer-${question.id}` ? "保存中..." : "保存回复"}
                          </Button>
                        </div>
                        <Textarea
                          rows={6}
                          value={draft.answer}
                          placeholder="填写对外展示给提问者的回复内容。"
                          onChange={(event) => handleDraftChange(question.id, "answer", event.target.value)}
                        />
                      </div>

                      <div className="space-y-3">
                        <div className="flex items-center justify-between">
                          <h2 className="text-sm font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">
                            内部备注
                          </h2>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() =>
                              void runMutation(
                                `note-${question.id}`,
                                () =>
                                  api.workspaces.questions.updateInternalNote(workspace.uid, question.id, {
                                    internal_note: drafts[question.id]?.internal_note || "",
                                  }),
                                "内部备注已更新。",
                                "保存内部备注失败，请稍后重试。"
                              )
                            }
                            disabled={busyKey === `note-${question.id}`}
                          >
                            {busyKey === `note-${question.id}` ? "保存中..." : "保存备注"}
                          </Button>
                        </div>
                        <Textarea
                          rows={6}
                          value={draft.internal_note}
                          placeholder="记录升级说明、内部沟通背景或处理建议。"
                          onChange={(event) => handleDraftChange(question.id, "internal_note", event.target.value)}
                        />
                      </div>
                    </div>

                    <div className="flex flex-wrap items-center justify-between gap-3 rounded-3xl bg-slate-50 px-4 py-3 dark:bg-slate-900">
                      <div className="text-sm text-slate-500 dark:text-slate-400">
                        {assignedMember ? `当前负责人：${assignedMember.name}` : "当前还没有负责人"}
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() =>
                          void runMutation(
                            `privacy-${question.id}`,
                            () =>
                              api.workspaces.questions.updatePrivacy(workspace.uid, question.id, {
                                is_private: !question.is_private,
                              }),
                            "问题可见性已更新。",
                            "更新问题可见性失败，请稍后重试。"
                          )
                        }
                        disabled={busyKey === `privacy-${question.id}`}
                      >
                        {question.is_private ? "改为公开" : "改为私密"}
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )}
      </div>
    </section>
  )
}

function StatCard({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: "slate" | "cyan" | "amber" | "emerald" | "violet"
}) {
  const toneClassName = {
    slate: "from-slate-100 to-white dark:from-slate-900 dark:to-slate-950",
    cyan: "from-cyan-100 to-white dark:from-cyan-950/50 dark:to-slate-950",
    amber: "from-amber-100 to-white dark:from-amber-950/40 dark:to-slate-950",
    emerald: "from-emerald-100 to-white dark:from-emerald-950/40 dark:to-slate-950",
    violet: "from-violet-100 to-white dark:from-violet-950/40 dark:to-slate-950",
  }[tone]

  return (
    <Card className={`border-slate-200 bg-gradient-to-br ${toneClassName} shadow-sm dark:border-slate-800`}>
      <CardContent className="pt-6">
        <p className="text-xs uppercase tracking-[0.24em] text-slate-500 dark:text-slate-400">{label}</p>
        <p className="mt-3 text-3xl font-semibold text-slate-950 dark:text-white">{value}</p>
      </CardContent>
    </Card>
  )
}
