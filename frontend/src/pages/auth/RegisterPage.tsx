import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { registerSchema, type RegisterFormValues } from '@/components/auth/validation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import ErrorMessage from '@/components/auth/ErrorMessage'
import LoadingSpinner from '@/components/auth/LoadingSpinner'
import { Label } from '@/components/ui/label'
import SocialLoginButton from '@/components/auth/SocialLoginButton'

export default function RegisterPage() {
  const navigate = useNavigate()
  const { useRegister } = useAuth()
  const registerMutation = useRegister()

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
  })

  const onSubmit = (data: RegisterFormValues) => {
    // Remove confirmPassword before sending to API
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { confirmPassword, ...apiData } = data
    registerMutation.mutate(apiData, {
      onSuccess: () => {
        navigate('/', { replace: true })
      },
    })
  }

  const hasFieldError = (fieldName: keyof RegisterFormValues) => {
    return errors[fieldName] !== undefined
  }

  const getFieldError = (fieldName: keyof RegisterFormValues) => {
    return errors[fieldName]?.message
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Create Account</CardTitle>
          <CardDescription>Sign up to get started with oReader</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {/* Social Login */}
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
                <Label htmlFor="nickname">Nickname (Optional)</Label>
                <Input
                  id="nickname"
                  type="text"
                  placeholder="Your display name"
                  {...register('nickname')}
                  aria-invalid={hasFieldError('nickname')}
                />
                {hasFieldError('nickname') && (
                  <p className="text-sm text-destructive">{getFieldError('nickname')}</p>
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

              <div className="space-y-2">
                <Label htmlFor="confirmPassword">Confirm Password</Label>
                <Input
                  id="confirmPassword"
                  type="password"
                  placeholder="••••••••"
                  {...register('confirmPassword')}
                  aria-invalid={hasFieldError('confirmPassword')}
                />
                {hasFieldError('confirmPassword') && (
                  <p className="text-sm text-destructive">{getFieldError('confirmPassword')}</p>
                )}
              </div>

              {registerMutation.error && <ErrorMessage message={registerMutation.error} />}

              <Button type="submit" className="w-full" disabled={registerMutation.isPending}>
                {registerMutation.isPending ? (
                  <>
                    <LoadingSpinner size="sm" />
                    <span className="ml-2">Creating account...</span>
                  </>
                ) : (
                  'Create Account'
                )}
              </Button>
            </form>
          </div>
        </CardContent>
        <CardFooter className="justify-center">
          <p className="text-sm text-muted-foreground">
            Already have an account?{' '}
            <Link to="/login" className="text-primary hover:underline">
              Sign in
            </Link>
          </p>
        </CardFooter>
      </Card>
    </div>
  )
}
