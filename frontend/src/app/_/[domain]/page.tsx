"use client"

import { useCallback, useEffect, useState } from "react"
import { useParams } from "next/navigation"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { Footer } from "@/components/layout/footer"
import { Header } from "@/components/layout/header"
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

    const response = await api.users.get(domain)
    setUser(response.user)
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

        setQuestions((current) => (cursor ? [...current, ...response.questions] : response.questions))
        setNextCursor(response.next_cursor || null)
        setHasMore(Boolean(response.next_cursor))
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

    let cancelled = false

    const bootstrap = async () => {
      setPageLoading(true)
      try {
        await Promise.all([loadPageUser(), loadQuestions()])
      } catch (error) {
        if (cancelled) {
          return
        }
        const message = error instanceof Error ? error.message : "加载提问箱失败，请稍后重试"
        toast(message, "error")
      } finally {
        if (!cancelled) {
          setPageLoading(false)
        }
      }
    }

    void bootstrap()

    return () => {
      cancelled = true
    }
  }, [domain, loadPageUser, loadQuestions, toast])

  const handleSubmitQuestion = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!domain?.trim()) {
      toast("提问箱地址无效，请刷新后重试", "error")
      return
    }

    const content = questionForm.content.trim()
    const receiveReplyEmail = notifyByEmail ? questionForm.receive_reply_email.trim() : ""

    if (!content) {
      toast("请输入提问内容", "warning")
      return
    }

    if (notifyByEmail && !receiveReplyEmail) {
      toast("请输入接收回复通知的邮箱", "warning")
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

      toast(response.message || "提问已发送", "success")
      setQuestionForm({
        content: "",
        is_private: false,
        receive_reply_email: "",
      })
      setNotifyByEmail(false)
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

  const coverStyle = user?.background
    ? {
        backgroundImage: `linear-gradient(135deg, rgba(37, 99, 235, 0.55), rgba(8, 145, 178, 0.35)), url(${user.background})`,
        backgroundPosition: "center",
        backgroundSize: "cover",
      }
    : undefined

  return (
    <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <Header />

      <div className="container mx-auto px-4 py-8">
        {pageLoading ? (
          <div className="flex min-h-[60vh] items-center justify-center">
            <div className="text-center">
              <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
              <p className="mt-4 text-gray-600 dark:text-gray-400">正在加载提问箱...</p>
            </div>
          </div>
        ) : !user ? (
          <div className="flex min-h-[60vh] items-center justify-center">
            <Card className="w-full max-w-xl text-center shadow-md">
              <CardContent className="py-12">
                <p className="text-lg font-medium text-gray-900 dark:text-gray-50">提问箱不存在</p>
                <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                  请确认访问的链接是否正确。
                </p>
              </CardContent>
            </Card>
          </div>
        ) : (
          <div className="mx-auto max-w-4xl space-y-6">
            <Card className="overflow-hidden border-0 shadow-xl">
              <div className="h-56 bg-gradient-to-br from-blue-600 via-sky-500 to-cyan-400" style={coverStyle} />
              <CardHeader className="relative -mt-16 pb-8 text-center">
                <Avatar className="mx-auto h-32 w-32 border-4 border-white shadow-2xl">
                  <AvatarImage src={user.avatar} alt={user.name} />
                  <AvatarFallback className="text-4xl">{user.name.charAt(0).toUpperCase()}</AvatarFallback>
                </Avatar>
                <h1 className="mt-4 text-3xl font-bold text-gray-900 dark:text-gray-50">{user.name}</h1>
                <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">@{user.domain}</p>
                {user.intro && (
                  <p className="mx-auto mt-4 max-w-2xl text-base leading-7 text-gray-600 dark:text-gray-300">
                    {user.intro}
                  </p>
                )}
              </CardHeader>
            </Card>

            <Card className="shadow-md">
              <CardContent className="pt-6">
                <div className="mb-5 text-center">
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    可以匿名提问，也可以选择在收到回复时通过邮箱获得提醒。
                  </p>
                </div>

                <form onSubmit={handleSubmitQuestion} className="space-y-4">
                  <div>
                    <Textarea
                      rows={5}
                      maxLength={1000}
                      className="resize-none"
                      placeholder="写下你想问的问题..."
                      value={questionForm.content}
                      onChange={(event) =>
                        setQuestionForm({ ...questionForm, content: event.target.value })
                      }
                    />
                    <p className="mt-1 text-right text-xs text-gray-500">{questionForm.content.length}/1000</p>
                  </div>

                  <label className="flex items-center gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm dark:border-gray-800 dark:bg-gray-900">
                    <input
                      type="checkbox"
                      checked={questionForm.is_private}
                      className="h-4 w-4 rounded border-gray-300"
                      onChange={(event) =>
                        setQuestionForm({ ...questionForm, is_private: event.target.checked })
                      }
                    />
                    <span>如果对方回答了，也不要在公开页面显示这条提问</span>
                  </label>

                  <div className="space-y-2">
                    <label className="flex items-center gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm dark:border-gray-800 dark:bg-gray-900">
                      <input
                        type="checkbox"
                        checked={notifyByEmail}
                        className="h-4 w-4 rounded border-gray-300"
                        onChange={(event) => {
                          const checked = event.target.checked
                          setNotifyByEmail(checked)
                          if (!checked) {
                            setQuestionForm((current) => ({ ...current, receive_reply_email: "" }))
                          }
                        }}
                      />
                      <span>收到回复时，通过邮箱通知我</span>
                    </label>

                    {notifyByEmail && (
                      <Input
                        type="email"
                        placeholder="输入接收通知的邮箱"
                        value={questionForm.receive_reply_email}
                        onChange={(event) =>
                          setQuestionForm({
                            ...questionForm,
                            receive_reply_email: event.target.value,
                          })
                        }
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
              <section className="space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-xl font-bold text-gray-900 dark:text-gray-50">公开回答</h2>
                  <Badge variant="secondary">{questions.length}</Badge>
                </div>

                {questions.map((question) => (
                  <Card key={question.id} className="shadow-md">
                    <CardContent className="pt-6">
                      <div className="mb-4">
                        <div className="mb-2 text-xs text-gray-500">{formatDate(question.created_at)}</div>
                        <p className="text-base leading-7 text-gray-900 dark:text-gray-100">{question.content}</p>
                      </div>

                      {question.answer && (
                        <>
                          <Separator className="my-4" />
                          <div className="rounded-2xl bg-blue-50 p-4 dark:bg-blue-900/20">
                            <p className="mb-2 text-sm font-medium text-blue-800 dark:text-blue-200">回答</p>
                            <p className="text-gray-900 dark:text-gray-100">{question.answer}</p>
                            <p className="mt-3 text-right text-xs text-gray-600 dark:text-gray-400">
                              来自 @{user.name} 的回复
                            </p>
                          </div>
                        </>
                      )}
                    </CardContent>
                  </Card>
                ))}

                {hasMore && (
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full"
                    onClick={() => void loadQuestions(nextCursor || undefined)}
                    disabled={loadingMore}
                  >
                    {loadingMore ? "加载中..." : "加载更多"}
                  </Button>
                )}
              </section>
            )}
          </div>
        )}
      </div>

      <Footer />
    </div>
  )
}
