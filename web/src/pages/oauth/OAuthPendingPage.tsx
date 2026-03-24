import { useState, useEffect } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { apiClient } from '../../lib/api/axios';

const emailSchema = z.object({
  email: z
    .string()
    .min(1, '请输入邮箱地址')
    .email('请输入有效的邮箱地址')
    .max(255, '邮箱地址不能超过255个字符'),
});

type EmailFormData = z.infer<typeof emailSchema>;

interface PendingOAuthInfo {
  github_login: string;
  nickname: string;
  avatar_url: string;
}

export function OAuthPendingPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [pendingInfo, setPendingInfo] = useState<PendingOAuthInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<EmailFormData>({
    resolver: zodResolver(emailSchema),
  });

  const token = searchParams.get('token');

  useEffect(() => {
    if (!token) {
      setError('无效的链接');
      setLoading(false);
      return;
    }

    const fetchPendingInfo = async () => {
      try {
        const response = await apiClient.get<PendingOAuthInfo>(
          `/auth/oauth/pending?token=${token}`
        );
        setPendingInfo(response.data);
      } catch {
        setError('链接已过期或无效，请重新登录');
      } finally {
        setLoading(false);
      }
    };

    fetchPendingInfo();
  }, [token]);

  const onSubmit = async (data: EmailFormData) => {
    if (!token) return;

    setSubmitting(true);
    setError(null);

    try {
      await apiClient.post('/auth/oauth/complete', {
        token,
        email: data.email,
      });
      navigate('/items');
    } catch (err: any) {
      const errorCode = err.response?.data?.error?.code;
      if (errorCode === 'EMAIL_ALREADY_USED') {
        setError('该邮箱已被注册，请使用其他邮箱');
      } else {
        setError('注册失败，请稍后重试');
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-lg text-gray-600">加载中...</div>
      </div>
    );
  }

  if (error && !pendingInfo) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <p className="text-lg text-red-600">{error}</p>
          <button
            onClick={() => navigate('/login')}
            className="mt-4 rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
          >
            返回登录
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-lg">
        <div className="mb-6 flex items-center justify-center gap-4">
          {pendingInfo?.avatar_url && (
            <img
              src={pendingInfo.avatar_url}
              alt="GitHub Avatar"
              className="h-16 w-16 rounded-full"
            />
          )}
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              欢迎来到 oReader！
            </h1>
          </div>
        </div>

        <div className="mb-6 rounded-lg bg-gray-50 p-4">
          <p className="text-sm text-gray-600">
            <strong>GitHub 账户:</strong> @{pendingInfo?.github_login}
          </p>
          {pendingInfo?.nickname && (
            <p className="mt-1 text-sm text-gray-600">
              <strong>昵称:</strong> {pendingInfo.nickname}
            </p>
          )}
        </div>

        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="mb-4">
            <label
              htmlFor="email"
              className="mb-2 block text-sm font-medium text-gray-700"
            >
              请提供您的邮箱地址：<span className="text-red-500">*</span>
            </label>
            <input
              id="email"
              type="email"
              {...register('email')}
              className="w-full rounded-lg border border-gray-300 px-4 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="your@email.com"
            />
            {errors.email && (
              <p className="mt-1 text-sm text-red-600">{errors.email.message}</p>
            )}
          </div>

          {error && (
            <div className="mb-4 rounded-lg bg-red-50 p-3 text-sm text-red-600">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-lg bg-blue-600 py-3 font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitting ? '处理中...' : '完成注册'}
          </button>
        </form>
      </div>
    </div>
  );
}
