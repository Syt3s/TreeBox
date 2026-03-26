export function Footer() {
  return (
    <footer className="border-t border-gray-200 bg-white py-6 dark:border-gray-800 dark:bg-gray-950">
      <div className="container mx-auto px-4">
        <div className="flex flex-col items-center justify-center gap-2 text-center text-sm text-gray-600 dark:text-gray-400">
          <p>
            Made with love by TreeBox
          </p>
          <div className="flex items-center gap-4">
            <a
              href="https://github.com/syt3s/TreeBox"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-gray-900 dark:hover:text-gray-100"
            >
              GitHub
            </a>
            <span>|</span>
            <a
              href="/pixel"
              className="hover:text-gray-900 dark:hover:text-gray-100"
            >
              鐢绘澘
            </a>
            <span>|</span>
            <a
              href="/sponsor"
              className="hover:text-gray-900 dark:hover:text-gray-100"
            >
              鎵撹祻
            </a>
          </div>
        </div>
      </div>
    </footer>
  )
}
