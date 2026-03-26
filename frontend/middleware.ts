import { NextRequest, NextResponse } from "next/server"

// Keep both cookie names during the TreeBox rename window.
const AUTH_TOKEN_COOKIE_NAMES = ["treebox_token", "nekobox_token"]

export function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl
  const token = AUTH_TOKEN_COOKIE_NAMES
    .map((name) => request.cookies.get(name)?.value?.trim() || "")
    .find(Boolean)

  if (pathname === "/login" || pathname === "/register") {
    if (token) {
      return NextResponse.redirect(new URL("/", request.url))
    }
    return NextResponse.next()
  }

  if (pathname === "/user" || pathname.startsWith("/user/")) {
    if (!token) {
      const loginUrl = new URL("/login", request.url)
      loginUrl.searchParams.set("to", `${pathname}${search}`)
      return NextResponse.redirect(loginUrl)
    }
  }

  return NextResponse.next()
}

export const config = {
  matcher: ["/login", "/register", "/user", "/user/:path*"],
}
