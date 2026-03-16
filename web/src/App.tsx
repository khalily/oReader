import { BrowserRouter, Routes, Route, Navigate, useSearchParams } from 'react-router-dom'
import { QueryClientProvider, QueryClient } from '@tanstack/react-query'
import { AppErrorBoundary } from '@/components/ErrorBoundary'
import { ToastProvider } from '@/components/ui/toast'
import { ThemeProvider } from '@/contexts/ThemeContext'
import ProtectedRoute from '@/components/ProtectedRoute'
import LoginPage from '@/pages/auth/LoginPage'
import RegisterPage from '@/pages/auth/RegisterPage'
import ItemsPage from '@/pages/items/ItemsPage'
import ItemViewPage from '@/pages/items/ItemViewPage'
import { useAuthStore } from '@/stores/authStore'
import { useEffect } from 'react'
import type { FilterType } from '@/components/feed/Sidebar'

// Create a client for React Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

// Wrapper component to handle URL params and filter/feed selection
function ItemsPageWrapper() {
  const [searchParams] = useSearchParams()
  const filter = searchParams.get('filter') as FilterType | null
  const feedId = searchParams.get('feed') || undefined

  return <ItemsPage filterType={filter ?? 'all'} feedId={feedId} />
}

function AppRoutes() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const initializeFromStorage = useAuthStore((state) => state.initializeFromStorage)

  // Initialize auth state from localStorage on mount
  useEffect(() => {
    initializeFromStorage()
  }, [initializeFromStorage])

  return (
    <Routes>
      {/* Public routes */}
      <Route
        path="/login"
        element={
          isAuthenticated ? (
            <Navigate to="/items" replace />
          ) : (
            <LoginPage />
          )
        }
      />
      <Route
        path="/register"
        element={
          isAuthenticated ? (
            <Navigate to="/items" replace />
          ) : (
            <RegisterPage />
          )
        }
      />

      {/* Protected routes */}
      <Route
        path="/items"
        element={
          <ProtectedRoute>
            <ItemsPageWrapper />
          </ProtectedRoute>
        }
      />
      <Route
        path="/items/:id"
        element={
          <ProtectedRoute>
            <ItemViewPage />
          </ProtectedRoute>
        }
      />

      {/* Default redirect */}
      <Route
        path="/"
        element={
          <Navigate to={isAuthenticated ? '/items' : '/login'} replace />
        }
      />

      {/* Catch all - redirect to items or login */}
      <Route
        path="*"
        element={
          <Navigate to={isAuthenticated ? '/items' : '/login'} replace />
        }
      />
    </Routes>
  )
}

function App() {
  return (
    <AppErrorBoundary
      onError={(error, errorInfo) => {
        // In production, you might send this to an error tracking service
        console.error('Application error:', error, errorInfo)
      }}
    >
      <ThemeProvider defaultTheme="system">
        <ToastProvider>
          <QueryClientProvider client={queryClient}>
            <BrowserRouter>
              <AppRoutes />
            </BrowserRouter>
          </QueryClientProvider>
        </ToastProvider>
      </ThemeProvider>
    </AppErrorBoundary>
  )
}

export default App
