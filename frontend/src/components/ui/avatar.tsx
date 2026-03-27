"use client"
/* eslint-disable @next/next/no-img-element */

import * as React from "react"
import { cn } from "@/lib/utils"

const Avatar = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full",
      className
    )}
    {...props}
  />
))
Avatar.displayName = "Avatar"

type AvatarImageProps = React.ImgHTMLAttributes<HTMLImageElement>

const AvatarImage = React.forwardRef<HTMLImageElement, AvatarImageProps>(
  ({ className, src, alt = "", ...props }, ref) => {
    const [hasError, setHasError] = React.useState(false)

    React.useEffect(() => {
      setHasError(false)
    }, [src])

    if (!src) {
      return null
    }

    if (typeof src === "string") {
      const trimmedSrc = src.trim()
      if (
        trimmedSrc === "" ||
        !trimmedSrc.startsWith("/") &&
        !trimmedSrc.startsWith("http://") &&
        !trimmedSrc.startsWith("https://") &&
        !trimmedSrc.startsWith("data:")
      ) {
        return null
      }
    }

    return (
      <img
        ref={ref}
        src={typeof src === "string" ? src : undefined}
        alt={alt}
        onError={() => setHasError(true)}
        className={cn("aspect-square h-full w-full", hasError && "hidden", className)}
        {...props}
      />
    )
  }
)
AvatarImage.displayName = "AvatarImage"

const AvatarFallback = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "flex h-full w-full items-center justify-center rounded-full bg-gray-100 dark:bg-gray-800",
      className
    )}
    {...props}
  />
))
AvatarFallback.displayName = "AvatarFallback"

export { Avatar, AvatarImage, AvatarFallback }
