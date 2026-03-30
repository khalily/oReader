import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate, useLocation, useSearchParams } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { loginSchema, type LoginFormValues } from '@/components/auth/validation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import ErrorMessage from '@/components/auth/ErrorMessage'
import LoadingSpinner from '@/components/auth/LoadingSpinner'
import { Label } from '@/components/ui/label'
import SocialLoginButton from '@/components/auth/SocialLoginButton'

// OAuth error messages
const oauthErrorMessages: Record<string, string> = {
  access_denied: 'GitHub login cancelled',
  invalid_state: 'Login expired, please try again',
  github_error: 'GitHub service temporarily unavailable',
}

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const { useLogin } = useAuth()
  const login = useLogin()

  const oauthError = searchParams.get('oauth_error')
  const oauthErrorMessage = oauthError ? oauthErrorMessages[oauthError] : null

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
  })

  const from = (location.state as { from?: { pathname: string } })?.from?.pathname || '/'

  const onSubmit = (data: LoginFormValues) => {
    login.mutate(data, {
      onSuccess: () => {
        navigate(from, { replace: true })
      },
    })
  }

  const hasFieldError = (fieldName: keyof LoginFormValues) => {
    return errors[fieldName] !== undefined
  }

  const getFieldError = (fieldName: keyof LoginFormValues) => {
    return errors[fieldName]?.message
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Sign In</CardTitle>
          <CardDescription>Enter your credentials to access your account</CardDescription>
        </CardHeader>
        <CardContent>
          {/* OAuth Error Message */}
          {oauthErrorMessage && (
            <div className="mb-4 rounded-md bg-destructive/10 p-3">
              <p className="text-sm text-destructive">{oauthErrorMessage}</p>
            </div>
          )}

          <div className="space-y-4">
            <SocialLoginButton provider="github" />

            {/* Divider */}
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-card px-2 text-muted-foreground">
                  or continue with
                </span>
              </div>
            </div>

            {/* Email/Password Form */}
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="your@email.com"
                  {...register('email')}
                  aria-invalid={hasFieldError('email')}
                />
                {hasFieldError('email') && (
                  <p className="text-sm text-destructive">{getFieldError('email')}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="••••••••"
                  {...register('password')}
                  aria-invalid={hasFieldError('password')}
                />
                {hasFieldError('password') && (
                  <p className="text-sm text-destructive">{getFieldError('password')}</p>
                )}
              </div>

              {login.error && <ErrorMessage message={login.error} />}

              <Button type="submit" className="w-full" disabled={login.isPending}>
                {login.isPending ? (
                  <>
                    <LoadingSpinner size="sm" />
                    <span className="ml-2">Signing in...</span>
                  </>
                ) : (
                  'Sign In'
                )}
              </Button>
            </form>
          </div>
        </CardContent>
        <CardFooter className="justify-center">
          <p className="text-sm text-muted-foreground">
            Don&apos;t have an account?{' '}
            <Link to="/register" className="text-primary hover:underline">
              Create one
            </Link>
          </p>
        </CardFooter>
      </Card>
    </div>
  )
}
