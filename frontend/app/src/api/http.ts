import axios, { AxiosError, type AxiosRequestConfig, type Method } from 'axios';

import { env } from '../config/env';

export type FieldErrors = Record<string, string[]>;

export class ApiError extends Error {
  status: number;
  code?: string;
  fieldErrors?: FieldErrors;

  constructor(message: string, status: number, code?: string, fieldErrors?: FieldErrors) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fieldErrors = fieldErrors;
  }
}

type ErrorResponse = {
  message?: string | FieldErrors;
  code?: string;
};

type ApiRequestOptions<TBody = unknown> = {
  method?: Method;
  data?: TBody;
  accessToken?: string | null;
  headers?: Record<string, string>;
};

// Axios instance used by the whole frontend.
// Keeping this centralized gives us one place for base URL, JSON behavior,
// interceptors, and future auth/cookie settings.
export const httpClient = axios.create({
  baseURL: env.apiBaseUrl,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Keep errors consistent for pages/components. UI code should catch ApiError and
// display `message`; if `code` exists, it can branch on stable backend codes.
function normalizeApiError(error: unknown): ApiError {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ErrorResponse>;
    const responseData = axiosError.response?.data;
    const message = responseData?.message;
    const fieldErrors = message && typeof message === 'object' ? message : undefined;

    return new ApiError(
      typeof message === 'string' ? message : axiosError.message || 'Request failed',
      axiosError.response?.status ?? 0,
      responseData?.code,
      fieldErrors,
    );
  }

  if (error instanceof Error) {
    return new ApiError(error.message, 0);
  }

  return new ApiError('Request failed', 0);
}

// Small wrapper around axios so feature modules do not directly depend on axios
// details. This makes later changes like cookies, interceptors, or retry logic
// much easier.
export async function apiRequest<TResponse, TBody = unknown>(
  path: string,
  options: ApiRequestOptions<TBody> = {},
): Promise<TResponse> {
  const config: AxiosRequestConfig<TBody> = {
    url: path,
    method: options.method ?? 'GET',
    data: options.data,
    headers: {
      ...options.headers,
    },
  };

  if (options.accessToken) {
    config.headers = {
      ...config.headers,
      Authorization: `Bearer ${options.accessToken}`,
    };
  }

  try {
    const response = await httpClient.request<TResponse>(config);
    return response.data;
  } catch (error) {
    throw normalizeApiError(error);
  }
}
