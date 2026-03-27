"use client"

import Image from "next/image"
import { Header } from "@/components/layout/header"
import { Footer } from "@/components/layout/footer"

export default function SponsorPage() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-amber-50 via-white to-sky-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <Header />

      <main className="container mx-auto px-4 py-16">
        <div className="mx-auto grid max-w-5xl gap-8 lg:grid-cols-[1.1fr_0.9fr]">
          <section className="space-y-6">
            <div className="inline-flex rounded-full border border-amber-200 bg-amber-100 px-4 py-1 text-sm font-medium text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/30 dark:text-amber-200">
              支持项目
            </div>
            <h1 className="text-4xl font-bold tracking-tight text-gray-900 dark:text-gray-100 md:text-5xl">
              给 TreeBox 打钱
            </h1>
            <p className="max-w-2xl text-lg leading-8 text-gray-600 dark:text-gray-400">
              TreeBox 最初只是一个轻量的匿名问答箱，后来慢慢变成了一个长期维护的小项目。如果你愿意分担一点服务器和开发成本，这里是支持入口。
            </p>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">为什么支持</h2>
                <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                  主要用于服务器、存储、图片上传和日常维护成本。
                </p>
              </div>
              <div className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">支持方式</h2>
                <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                  当前保留的是支付宝二维码，后续可以继续扩展更多方式。
                </p>
              </div>
            </div>
          </section>

          <aside className="rounded-[2rem] border border-gray-200 bg-white p-6 shadow-xl dark:border-gray-800 dark:bg-gray-900">
            <div className="rounded-[1.5rem] bg-sky-500 px-5 py-4 text-white shadow-lg">
              <p className="text-center text-sm font-medium opacity-90">谢谢老板 Thanks♪(･ω･)ﾉ</p>
              <div className="mt-4 overflow-hidden rounded-2xl bg-white p-4">
                <Image
                  src="https://img13.360buyimg.com/ceco/jfs/t1/2848/30/22668/22212/66b6ee87F15f75426/9f8969181dd1f244.png"
                  alt="支付宝收款二维码"
                  width={280}
                  height={280}
                  unoptimized
                  className="mx-auto h-auto w-full max-w-[280px] rounded-xl"
                />
              </div>
            </div>
            <div className="mt-6 space-y-4">
              <p className="text-sm leading-6 text-gray-600 dark:text-gray-400">
                感谢所有支持 TreeBox 的朋友。你的每一次支持，都会直接变成项目维护的时间和资源。
              </p>
              <div className="rounded-2xl bg-amber-50 p-4 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
                当前页面是前端实现，旧的后端模板页已经移除。
              </div>
            </div>
          </aside>
        </div>
      </main>

      <Footer />
    </div>
  )
}
