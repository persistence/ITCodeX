import { useEffect, useMemo, useState } from 'react'
import { App, ConfigProvider, theme as antdTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/layouts/AppLayout'
import { AppRouter } from '@/router'
import { getSettings, resolveTheme, type AppSettings } from '@/lib/settings'
import 'dayjs/locale/zh-cn'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 10_000,
    },
  },
})

export function RootApp() {
  const [settings, setSettings] = useState<AppSettings>(() => getSettings())
  const mode = resolveTheme(settings.theme)

  useEffect(() => {
    const onSettings = (e: Event) => {
      setSettings((e as CustomEvent<AppSettings>).detail)
    }
    window.addEventListener('itcodex:settings', onSettings)
    return () => window.removeEventListener('itcodex:settings', onSettings)
  }, [])

  const algorithm = useMemo(
    () => (mode === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm),
    [mode],
  )

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm,
        token: {
          colorPrimary: '#2f6fed',
          borderRadius: 10,
          fontFamily:
            '"IBM Plex Sans", "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
        },
      }}
    >
      <App>
        <QueryClientProvider client={queryClient}>
          <BrowserRouter>
            <AppLayout>
              <AppRouter />
            </AppLayout>
          </BrowserRouter>
        </QueryClientProvider>
      </App>
    </ConfigProvider>
  )
}
