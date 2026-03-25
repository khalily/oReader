import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { apiClient, setCsrfToken } from '@/lib/api/axios';
import { useAuthStore } from '@/stores/authStore';
import type { User } from '@/types';

interface OAuthResponse {
  user: User;
  csrf_token: string;
  redirect?: string; // 'pending' for email collection flow
  pending_token?: string; // Token for pending OAuth
}

export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [processing, setProcessing] = useState(true);
  const setUser = useAuthStore((state) => state.setUser);

  useEffect(() => {
    const code = searchParams.get('code');
    const state = searchParams.get('state');
    const errorParam = searchParams.get('error');
    const errorDesc = searchParams.get('error_description');

    // Handle OAuth error from GitHub
    if (errorParam) {
      setError(errorDesc || 'GitHub 授权失败');
      setProcessing(false);
      return;
    }

    // Missing required parameters
    if (!code || !state) {
      setError('无效的回调参数');
      setProcessing(false);
      return;
    }

    const completeOAuth = async () => {
      try {
        // Call backend API with format=json to get JSON response instead of 302
        const response = await apiClient.get<OAuthResponse>(
          `/auth/github/callback`,
          {
            params: {
              code,
              state,
              format: 'json',
            },
            // Don't follow redirects - we want to handle them manually
            maxRedirects: 0,
          }
        );

        const data = response.data;

        // Check if we need to redirect to pending page (no email case)
        if (data.redirect === 'pending' && data.pending_token) {
          navigate(`/oauth/pending?token=${data.pending_token}`);
          return;
        }

        // Success - set auth state
        setUser(data.user, data.csrf_token);
        setCsrfToken(data.csrf_token);

        // Navigate to items page
        navigate('/items', { replace: true });
      } catch (err: any) {
        console.error('OAuth callback error:', err);

        // Handle specific error cases
        if (err.response?.status === 401) {
          setError('授权已过期，请重新登录');
        } else if (err.response?.data?.error) {
          setError(err.response.data.error);
        } else {
          setError('登录失败，请稍后重试');
        }
      } finally {
        setProcessing(false);
      }
    };

    completeOAuth();
  }, [searchParams, navigate, setUser]);

  if (processing) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="mb-4 h-12 w-12 animate-spin rounded-full border-4 border-blue-600 border-t-transparent mx-auto"></div>
          <p className="text-lg text-gray-600">正在完成 GitHub 登录...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-lg text-center">
          <div className="mb-4 text-red-500">
            <svg
              className="h-12 w-12 mx-auto"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </div>
          <h2 className="mb-2 text-xl font-semibold text-gray-900">登录失败</h2>
          <p className="mb-6 text-gray-600">{error}</p>
          <button
            onClick={() => navigate('/login')}
            className="rounded-lg bg-blue-600 px-6 py-2 text-white hover:bg-blue-700"
          >
            返回登录
          </button>
        </div>
      </div>
    );
  }

  return null;
}
