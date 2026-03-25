"use client"

import { useCallback, useEffect, useState } from "react"
import { useParams } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Header } from "@/components/layout/header"
import { Footer } from "@/components/layout/footer"
import { useToast } from "@/components/ui/toast"
import { api } from "@/lib/api"
import type { PublicUser, Question } from "@/types"

export default function UserBoxPage() {
  const params = useParams()
  const domainParam = params.domain
  const domain = Array.isArray(domainParam) ? domainParam[0] : domainParam
  const { toast } = useToast()

  const [user, setUser] = useState<PublicUser | null>(null)
  const [questions, setQuestions] = useState<Question[]>([])
  const [pageLoading, setPageLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(true)
  const [notifyByEmail, setNotifyByEmail] = useState(false)
  const [questionForm, setQuestionForm] = useState({
    content: "",
    is_private: false,
    receive_reply_email: "",
  })

  const loadPageUser = useCallback(async () => {
    if (!domain?.trim()) {
      return
    }

    const res = await api.users.get(domain)
    if (res.success) {
      setUser(res.user)
    }
  }, [domain])

  const loadQuestions = useCallback(
    async (cursor?: string) => {
      if (!domain?.trim()) {
        return
      }

      if (cursor) {
        setLoadingMore(true)
      }

      try {
        const response = await api.questions.list(domain, {
          page_size: 20,
          cursor,
        })

        if (response.success) {
          setQuestions((prev) => (cursor ? [...prev, ...response.questions] : response.questions))
          setNextCursor(response.next_cursor || null)
          setHasMore(Boolean(response.next_cursor))
        }
      } finally {
        setLoadingMore(false)
      }
    },
    [domain]
  )

  useEffect(() => {
    if (!domain?.trim()) {
      return
    }

    let active = true

    const bootstrap = async () => {
      setPageLoading(true)
      try {
        await Promise.all([loadPageUser(), loadQuestions()])
      } catch (error) {
        if (!active) {
          return
        }
        const message = error instanceof Error ? error.message : "加载页面失败，请稍后重试"
        toast(message, "error")
      } finally {
        if (active) {
          setPageLoading(false)
        }
      }
    }

    void bootstrap()

    return () => {
      active = false
    }
  }, [domain, loadPageUser, loadQuestions, toast])

  const handleSubmitQuestion = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!domain?.trim()) {
      toast("提问箱地址无效，请刷新后重试", "error")
      return
    }

    const content = questionForm.content.trim()
    const receiveReplyEmail = notifyByEmail ? questionForm.receive_reply_email.trim() : ""

    if (!content) {
      toast("请输入问题内容", "warning")
      return
    }
    if (notifyByEmail && !receiveReplyEmail) {
      toast("请输入用于接收回复通知的邮箱", "warning")
      return
    }

    setSubmitting(true)

    try {
      const response = await api.questions.create(domain, {
        content,
        is_private: questionForm.is_private,
        receive_reply_email: receiveReplyEmail,
        recaptcha: "test",
      })

      if (response.success) {
        toast(response.message || "提问成功", "success")
        setQuestionForm({
          content: "",
          is_private: false,
          receive_reply_email: "",
        })
        setNotifyByEmail(false)
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "提问失败，请稍后重试"
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

  return (
    <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <Header />

      <div className="container mx-auto px-4 py-8">
        {pageLoading ? (
          <div className="flex min-h-[60vh] items-center justify-center">
            <div className="text-center">
              <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
              <p className="mt-4 text-gray-600 dark:text-gray-400">加载中...</p>
            </div>
          </div>
        ) : (
          <div className="mx-auto max-w-3xl space-y-6">
            <Card className="overflow-hidden shadow-lg">
              <div className="h-48 bg-gradient-to-br from-blue-600 to-cyan-500" />
              <CardHeader className="relative -mt-16 text-center">
                <Avatar className="mx-auto h-32 w-32 border-4 border-white shadow-lg">
                  <AvatarImage src={user?.avatar || "/default-avatar.png"} alt={user?.name || "User"} />
                  <AvatarFallback className="text-4xl">
                    {user?.name?.charAt(0).toUpperCase() || "U"}
                  </AvatarFallback>
                </Avatar>
                <h1 className="mt-4 text-3xl font-bold text-gray-900 dark:text-gray-100">{user?.name || domain}</h1>
                {user?.intro && <p className="mt-2 text-gray-600 dark:text-gray-400">{user.intro}</p>}
              </CardHeader>
            </Card>

            <Card className="shadow-md">
              <CardContent className="pt-6">
                <div className="mb-4 text-center">
                  <p className="text-sm text-gray-600 dark:text-gray-400">任何人都可以匿名向这个提问箱发送问题。</p>
                </div>
                <form onSubmit={handleSubmitQuestion} className="space-y-4">
                  <div>
                    <Textarea
                      placeholder="写下你想问的问题..."
                      value={questionForm.content}
                      onChange={(e) => setQuestionForm({ ...questionForm, content: e.target.value })}
                      rows={5}
                      maxLength={1000}
                      className="resize-none"
                    />
                    <p className="mt-1 text-right text-xs text-gray-500">{questionForm.content.length}/1000</p>
                  </div>

                  <div className="space-y-2">
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={questionForm.is_private}
                        onChange={(e) => setQuestionForm({ ...questionForm, is_private: e.target.checked })}
                        className="h-4 w-4 rounded border-gray-300"
                      />
                      <span>回答后不公开显示这条提问</span>
                    </label>
                  </div>

                  <div className="space-y-2">
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={notifyByEmail}
                        onChange={(e) => {
                          const checked = e.target.checked
                          setNotifyByEmail(checked)
                          if (!checked) {
                            setQuestionForm((current) => ({ ...current, receive_reply_email: "" }))
                          }
                        }}
                        className="h-4 w-4 rounded border-gray-300"
                      />
                      <span>我想在收到回复时获取邮件通知</span>
                    </label>
                    {notifyByEmail && (
                      <Input
                        type="email"
                        placeholder="输入接收通知的邮箱"
                        value={questionForm.receive_reply_email}
                        onChange={(e) => setQuestionForm({ ...questionForm, receive_reply_email: e.target.value })}
                      />
                    )}
                  </div>

                  <Button type="submit" className="w-full" disabled={submitting}>
                    {submitting ? "发送中..." : "发送提问"}
                  </Button>
                </form>
              </CardContent>
            </Card>

            {questions.length > 0 && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-xl font-bold text-gray-900 dark:text-gray-100">已公开回答的问题</h2>
                  <Badge variant="secondary">{questions.length}</Badge>
                </div>

                {questions.map((question) => (
                  <Card key={question.id} className="shadow-md">
                    <CardContent className="pt-6">
                      <div className="mb-4">
                        <div className="mb-2 text-xs text-gray-500">{formatDate(question.created_at)}</div>
                        <p className="text-gray-900 dark:text-gray-100">{question.content}</p>
                      </div>

                      {question.answer && (
                        <>
                          <Separator className="my-4" />
                          <div className="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
                            <p className="mb-2 text-sm font-medium text-blue-800 dark:text-blue-200">回答</p>
                            <p className="text-gray-900 dark:text-gray-100">{question.answer}</p>
                            <p className="mt-2 text-right text-xs text-gray-600 dark:text-gray-400">
                              来自 @{user?.name || domain} 的回答
                            </p>
                          </div>
                        </>
                      )}
                    </CardContent>
                  </Card>
                ))}

                {hasMore && (
                  <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => void loadQuestions(nextCursor || undefined)}
                    disabled={loadingMore}
                  >
                    {loadingMore ? "加载中..." : "加载更多"}
                  </Button>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      <Footer />
    </div>
  )
}
