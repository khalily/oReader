import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClientProvider, QueryClient } from '@tanstack/react-query'
import { AppErrorBoundary } from '@/components/ErrorBoundary'
import { ToastProvider } from '@/components/ui/toast'
import { ThemeProvider } from '@/contexts/ThemeContext'
import ProtectedRoute from '@/components/ProtectedRoute'
import LoginPage from '@/pages/auth/LoginPage'
import RegisterPage from '@/pages/auth/RegisterPage'
import { OAuthPendingPage } from '@/pages/oauth/OAuthPendingPage'
import { OAuthCallbackPage } from '@/pages/oauth/OAuthCallbackPage'
import ItemsPage from '@/pages/items/ItemsPage'
import ItemViewPage from '@/pages/items/ItemViewPage'
import PaperViewPage from '@/pages/papers/PaperViewPage'
import { useAuthStore } from '@/stores/authStore'
import { useTokenRefresh } from '@/hooks/useTokenRefresh'
import { useEffect } from 'react'

// Create a client for React Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

function AppRoutes() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const initializeFromStorage = useAuthStore((state) => state.initializeFromStorage)

  // Proactive token refresh — keeps access token alive before it expires
  useTokenRefresh()

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
            <Navigate to="/feeds" replace />
          ) : (
            <LoginPage />
          )
        }
      />
      <Route
        path="/register"
        element={
          isAuthenticated ? (
            <Navigate to="/feeds" replace />
          ) : (
            <RegisterPage />
          )
        }
      />
      <Route
        path="/oauth/pending"
        element={
          isAuthenticated ? (
            <Navigate to="/feeds" replace />
          ) : (
            <OAuthPendingPage />
          )
        }
      />
      {/* OAuth callback - handles GitHub OAuth redirect */}
      <Route
        path="/api/v1/auth/github/callback"
        element={<OAuthCallbackPage />}
      />

      {/* Protected routes */}
      <Route
        path="/feeds"
        element={
          <ProtectedRoute>
            <ItemsPage />
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
      <Route
        path="/papers/:id"
        element={
          <ProtectedRoute>
            <PaperViewPage />
          </ProtectedRoute>
        }
      />

      {/* Legacy redirects */}
      <Route
        path="/items"
        element={<Navigate to="/feeds" replace />}
      />
      <Route
        path="/papers"
        element={<Navigate to="/feeds" replace />}
      />

      {/* Default redirect */}
      <Route
        path="/"
        element={
          <Navigate to={isAuthenticated ? '/feeds' : '/login'} replace />
        }
      />

      {/* Catch all - redirect to feeds or login */}
      <Route
        path="*"
        element={
          <Navigate to={isAuthenticated ? '/feeds' : '/login'} replace />
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
