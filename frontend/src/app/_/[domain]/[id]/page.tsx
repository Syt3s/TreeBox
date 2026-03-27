"use client"

import { useCallback, useEffect, useState } from "react"
import { useParams, useRouter } from "next/navigation"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { Footer } from "@/components/layout/footer"
import { Header } from "@/components/layout/header"
import { useToast } from "@/components/ui/toast"
import { api } from "@/lib/api"
import { emitQuestionStatsRefresh } from "@/lib/question-stats"
import type { PublicUser, Question } from "@/types"

export default function QuestionDetailPage() {
  const params = useParams()
  const router = useRouter()
  const domainParam = params.domain
  const idParam = params.id
  const domain = Array.isArray(domainParam) ? domainParam[0] : domainParam
  const questionId = Number(Array.isArray(idParam) ? idParam[0] : idParam)
  const { toast } = useToast()

  const [user, setUser] = useState<PublicUser | null>(null)
  const [question, setQuestion] = useState<Question | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [canDelete, setCanDelete] = useState(false)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [answerForm, setAnswerForm] = useState({ answer: "" })

  const loadQuestion = useCallback(async () => {
    if (!domain?.trim() || !Number.isFinite(questionId) || questionId <= 0) {
      return
    }

    const response = await api.questions.get(domain, questionId)
    if (response.question) {
      let nextQuestion = response.question

      if (response.can_delete && !response.question.viewed_at) {
        try {
          const viewedResponse = await api.user.questions.markViewed(questionId)
          nextQuestion = {
            ...response.question,
            viewed_at: viewedResponse.viewed_at || new Date().toISOString(),
          }
          emitQuestionStatsRefresh()
        } catch {
          nextQuestion = response.question
        }
      }

      setQuestion(nextQuestion)
      setCanDelete(Boolean(response.can_delete))
      setAnswerForm({ answer: nextQuestion.answer || "" })
    } else {
      setQuestion(null)
    }
  }, [domain, questionId])

  const loadPageUser = useCallback(async () => {
    if (!domain?.trim()) {
      return
    }

    const response = await api.users.get(domain)
    setUser(response.user)
  }, [domain])

  useEffect(() => {
    if (!domain?.trim() || !Number.isFinite(questionId) || questionId <= 0) {
      return
    }

    let cancelled = false

    const bootstrap = async () => {
      setLoading(true)
      try {
        await Promise.all([loadPageUser(), loadQuestion()])
      } catch (error) {
        if (cancelled) {
          return
        }
        const message = error instanceof Error ? error.message : "加载问题失败，请稍后重试"
        toast(message, "error")
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
  }, [domain, questionId, loadPageUser, loadQuestion, toast])

  const handleSubmitAnswer = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!domain?.trim() || !Number.isFinite(questionId) || questionId <= 0) {
      toast("问题地址无效，请刷新后重试", "error")
      return
    }

    const answer = answerForm.answer.trim()
    if (!answer) {
      toast("请输入回答内容", "warning")
      return
    }

    setSubmitting(true)

    try {
      const response = await api.questions.answer(domain, questionId, { answer })
      toast(response.message || "回答已发布", "success")
      await loadQuestion()
      emitQuestionStatsRefresh()
    } catch (error) {
      const message = error instanceof Error ? error.message : "回答失败，请稍后重试"
      toast(message, "error")
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!domain?.trim() || !Number.isFinite(questionId) || questionId <= 0) {
      toast("问题地址无效，请刷新后重试", "error")
      return
    }

    setSubmitting(true)

    try {
      const response = await api.questions.delete(domain, questionId)
      toast(response.message || "提问已删除", "success")
      emitQuestionStatsRefresh()
      router.push(`/user/questions`)
    } catch (error) {
      const message = error instanceof Error ? error.message : "删除失败，请稍后重试"
      toast(message, "error")
    } finally {
      setSubmitting(false)
      setShowDeleteDialog(false)
    }
  }

  const handleSetPrivate = async () => {
    if (!domain?.trim() || !Number.isFinite(questionId) || questionId <= 0) {
      toast("问题地址无效，请刷新后重试", "error")
      return
    }

    setSubmitting(true)

    try {
      const response = await api.questions.setPrivate(domain, questionId)
      toast(response.message || "已设为私密", "success")
      await loadQuestion()
    } catch (error) {
      const message = error instanceof Error ? error.message : "设置失败，请稍后重试"
      toast(message, "error")
    } finally {
      setSubmitting(false)
    }
  }

  const handleSetPublic = async () => {
    if (!domain?.trim() || !Number.isFinite(questionId) || questionId <= 0) {
      toast("问题地址无效，请刷新后重试", "error")
      return
    }

    setSubmitting(true)

    try {
      const response = await api.questions.setPublic(domain, questionId)
      toast(response.message || "已设为公开", "success")
      await loadQuestion()
    } catch (error) {
      const message = error instanceof Error ? error.message : "设置失败，请稍后重试"
      toast(message, "error")
    } finally {
      setSubmitting(false)
    }
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
        <Header />
        <div className="flex min-h-[60vh] items-center justify-center">
          <div className="text-center">
            <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
            <p className="mt-4 text-gray-600 dark:text-gray-400">正在加载问题...</p>
          </div>
        </div>
        <Footer />
      </div>
    )
  }

  if (!question) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
        <Header />
        <div className="flex min-h-[60vh] items-center justify-center px-4">
          <Card className="w-full max-w-xl text-center shadow-md">
            <CardContent className="py-12">
              <p className="text-lg font-medium text-gray-900 dark:text-gray-50">
                这个问题不存在，或者你没有权限查看。
              </p>
              <Button type="button" variant="outline" className="mt-4" onClick={() => router.push(`/box/${domain}`)}>
                返回提问箱
              </Button>
            </CardContent>
          </Card>
        </div>
        <Footer />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <Header />

      <div className="container mx-auto px-4 py-8">
        <div className="mx-auto max-w-3xl space-y-6">
          <Button type="button" variant="ghost" onClick={() => router.push(`/box/${domain}`)}>
            返回提问箱
          </Button>

          <Card className="shadow-lg">
            <CardHeader>
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="flex items-center gap-3">
                  <Avatar>
                    <AvatarImage src={user?.avatar} alt={user?.name || domain} />
                    <AvatarFallback>{user?.name?.charAt(0).toUpperCase() || "U"}</AvatarFallback>
                  </Avatar>
                  <div>
                    <CardTitle className="text-lg">{user?.name || domain}</CardTitle>
                    <p className="text-xs text-gray-500">{formatDate(question.created_at)}</p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {!question.answer && (
                    <Badge
                      variant="outline"
                      className="border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300"
                    >
                      待回答
                    </Badge>
                  )}
                  {question.is_private && <Badge variant="outline">私密</Badge>}
                </div>
              </div>
            </CardHeader>

            <CardContent>
              <p className="text-lg leading-8 text-gray-900 dark:text-gray-100">{question.content}</p>

              {question.answer && (
                <>
                  <Separator className="my-6" />
                  <div className="rounded-2xl bg-sky-50 p-6 dark:bg-sky-950/30">
                    <p className="mb-3 text-sm font-medium text-sky-800 dark:text-sky-200">回答</p>
                    <p className="text-gray-900 dark:text-gray-100">{question.answer}</p>
                    <p className="mt-3 text-right text-sm text-gray-600 dark:text-gray-400">
                      来自 @{user?.name || domain} 的回复
                    </p>
                  </div>
                </>
              )}

              {canDelete && (
                <div className="mt-6 flex flex-wrap items-center justify-end gap-2">
                  {question.is_private ? (
                    <Button type="button" variant="outline" size="sm" onClick={handleSetPublic} disabled={submitting}>
                      设为公开
                    </Button>
                  ) : (
                    <Button type="button" variant="outline" size="sm" onClick={handleSetPrivate} disabled={submitting}>
                      设为私密
                    </Button>
                  )}
                  <Button type="button" variant="outline" size="sm" onClick={() => setShowDeleteDialog(true)}>
                    删除提问
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>

          {canDelete && (
            <Card className="shadow-md">
              <CardHeader>
                <CardTitle>{question.answer ? "编辑回答" : "回答问题"}</CardTitle>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleSubmitAnswer} className="space-y-4">
                  <div>
                    <Textarea
                      rows={6}
                      maxLength={1000}
                      className="resize-none"
                      placeholder="写下你的回答..."
                      value={answerForm.answer}
                      onChange={(event) => setAnswerForm({ answer: event.target.value })}
                    />
                    <p className="mt-1 text-right text-xs text-gray-500">{answerForm.answer.length}/1000</p>
                  </div>
                  <Button type="submit" disabled={submitting}>
                    {submitting ? "提交中..." : question.answer ? "更新回答" : "发布回答"}
                  </Button>
                </form>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除这条提问？</DialogTitle>
            <DialogDescription>删除后无法恢复，请谨慎操作。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setShowDeleteDialog(false)}>
              取消
            </Button>
            <Button type="button" onClick={handleDelete} disabled={submitting}>
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Footer />
    </div>
  )
}
