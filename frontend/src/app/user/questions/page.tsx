"use client"

import { useCallback, useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { useToast } from "@/components/ui/toast"
import { useAuth } from "@/components/auth/auth-provider"
import { api } from "@/lib/api"
import { emitQuestionStatsRefresh } from "@/lib/question-stats"
import type { Question } from "@/types"

export default function UserQuestionsPage() {
  const router = useRouter()
  const { toast } = useToast()
  const { user, loading: authLoading } = useAuth()

  const [questions, setQuestions] = useState<Question[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(true)

  const loadQuestions = useCallback(
    async (cursor?: string) => {
      if (!user) {
        return
      }

      if (cursor) {
        setLoadingMore(true)
      } else {
        setLoading(true)
      }

      try {
        const response = await api.user.questions.list({
          page_size: 20,
          cursor,
        })

        setQuestions((current) => (cursor ? [...current, ...response.questions] : response.questions))
        setNextCursor(response.next_cursor || null)
        setHasMore(Boolean(response.next_cursor))
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [user]
  )

  const markAllQuestionsViewed = useCallback(async () => {
    if (!user) {
      return
    }

    const response = await api.user.questions.markAllViewed()
    if (!response.viewed_count) {
      return
    }

    setQuestions((current) =>
      current.map((question) =>
        question.viewed_at
          ? question
          : {
              ...question,
              viewed_at: response.viewed_at || new Date().toISOString(),
            }
      )
    )
    emitQuestionStatsRefresh()
  }, [user])

  useEffect(() => {
    if (authLoading) {
      return
    }

    if (!user) {
      router.replace("/login?to=/user/questions")
      return
    }

    let cancelled = false

    const bootstrap = async () => {
      try {
        await loadQuestions()
        if (!cancelled) {
          await markAllQuestionsViewed()
        }
      } catch (error) {
        if (cancelled) {
          return
        }
        const message = error instanceof Error ? error.message : "加载问题列表失败，请稍后重试"
        toast(message, "error")
      }
    }

    void bootstrap()

    return () => {
      cancelled = true
    }
  }, [authLoading, loadQuestions, markAllQuestionsViewed, router, toast, user])

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

  const handleOpenQuestion = (questionId: number) => {
    if (!user) {
      return
    }

    router.push(`/box/${user.domain}/${questionId}`)
  }

  if (authLoading || loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
          <p className="mt-4 text-gray-600 dark:text-gray-400">正在加载问题...</p>
        </div>
      </div>
    )
  }

  if (!user) {
    return null
  }

  return (
    <section className="container mx-auto px-4 py-8">
      <div className="mx-auto max-w-5xl space-y-6">
        <Card className="shadow-lg">
          <CardHeader>
            <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <CardTitle>问题列表</CardTitle>
              </div>
              <Button type="button" variant="outline" onClick={() => router.push(`/box/${user.domain}`)}>
                查看我的提问箱
              </Button>
            </div>
          </CardHeader>
        </Card>

        {questions.length === 0 ? (
          <Card className="shadow-md">
            <CardContent className="py-14 text-center">
              <p className="text-lg font-medium text-gray-900 dark:text-gray-50">还没有收到问题</p>
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                分享你的提问箱链接后，新的提问会出现在这里。
              </p>
              <Button
                type="button"
                variant="outline"
                className="mt-5"
                onClick={() => router.push(`/box/${user.domain}`)}
              >
                去看看我的提问箱
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            {questions.map((question) => {
              const answered = Boolean(question.answer)

              return (
                <Card key={question.id} className="shadow-md">
                  <CardContent className="pt-6">
                    <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                      <div className="space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="text-xs text-gray-500">{formatDate(question.created_at)}</span>
                          {answered ? (
                            <Badge variant="secondary">已回答</Badge>
                          ) : (
                            <Badge
                              variant="outline"
                              className="border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300"
                            >
                              待回答
                            </Badge>
                          )}
                          {question.is_private && <Badge variant="outline">私密</Badge>}
                        </div>
                        <p className="text-base leading-7 text-gray-900 dark:text-gray-100">{question.content}</p>
                      </div>

                      <Button
                        type="button"
                        variant={answered ? "outline" : "default"}
                        onClick={() => handleOpenQuestion(question.id)}
                      >
                        {answered ? "查看或编辑回答" : "去回答"}
                      </Button>
                    </div>

                    {answered && question.answer && (
                      <>
                        <Separator className="my-4" />
                        <div className="rounded-2xl bg-sky-50 p-4 dark:bg-sky-950/30">
                          <p className="mb-2 text-sm font-medium text-sky-800 dark:text-sky-200">我的回答</p>
                          <p className="text-gray-900 dark:text-gray-100">{question.answer}</p>
                        </div>
                      </>
                    )}
                  </CardContent>
                </Card>
              )
            })}

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
          </div>
        )}
      </div>
    </section>
  )
}
