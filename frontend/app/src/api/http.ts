import axios, {
  type AxiosError,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
  type Method,
} from 'axios';

import { env } from '../config/env';
import { readAccessToken, refreshForRetry } from './authTokenManager';

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

// Extension on InternalAxiosRequestConfig for our internal retry marker. This
// property is set by the REQUEST interceptor (so it's on the final merged
// config axios actually uses) when it detects our retry header. That way we
// reliably know, in the response interceptor, whether this request was a
// retry — even though axios clones configs.
interface DrawoConfig {
  __drawoRetried?: boolean;
  __skipAuth?: boolean;
}

// Public/internal header used to tag retried requests. It is stripped before
// the request leaves the browser. Using a header (instead of a property on
// the config object) guarantees the flag survives axios's config merging and
// cloning between our .request() call and the adapter invocation.
const RETRY_HEADER = 'X-Drawo-Retry';

export const httpClient = axios.create({
  baseURL: env.apiBaseUrl,
  timeout: 15_000, // never hang forever on a dead/unreachable backend
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor:
//  1. Detect & strip the internal retry header and mark the config.
//  2. Opt out of auth when caller passed accessToken: null.
//  3. Auto-attach the Bearer token from the auth store.
httpClient.interceptors.request.use((config: InternalAxiosRequestConfig & DrawoConfig) => {
  // Detect the retry flag sent via header (survives axios config cloning).
  if (config.headers.has(RETRY_HEADER)) {
    config.__drawoRetried = true;
    config.headers.delete(RETRY_HEADER);
  }

  if (config.__skipAuth) return config;

  const token = readAccessToken();
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`);
  }
  return config;
});

// Response interceptor: reactive token refresh on 401 with single-retry limit.
httpClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ErrorResponse>) => {
    const config = error.config as (InternalAxiosRequestConfig & DrawoConfig) | undefined;
    const status = error.response?.status;

    if (status !== 401 || !config) {
      return Promise.reject(normalizeApiError(error));
    }
    if (config.__drawoRetried) {
      // Already retried once — give up.
      return Promise.reject(normalizeApiError(error));
    }
    if (config.__skipAuth) {
      return Promise.reject(normalizeApiError(error));
    }
    if (!config.headers.Authorization) {
      return Promise.reject(normalizeApiError(error));
    }

    try {
      const newToken = await refreshForRetry();
      // Tag the retried request via header (the request interceptor will
      // translate this into __drawoRetried = true and strip the header).
      config.headers.set(RETRY_HEADER, '1');
      config.headers.set('Authorization', `Bearer ${newToken}`);
      return httpClient.request(config);
    } catch {
      return Promise.reject(normalizeApiError(error));
    }
  },
);

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

export async function apiRequest<TResponse, TBody = unknown>(
  path: string,
  options: ApiRequestOptions<TBody> = {},
): Promise<TResponse> {
  const config: AxiosRequestConfig<TBody> & DrawoConfig = {
    url: path,
    method: options.method ?? 'GET',
    data: options.data,
    headers: {
      ...options.headers,
    },
  };

  if (options.accessToken === null) {
    config.__skipAuth = true;
  } else if (options.accessToken) {
    config.headers = {
      ...config.headers,
      Authorization: `Bearer ${options.accessToken}`,
    };
  }

  try {
    const response = await httpClient.request<TResponse>(config);
    return response.data;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    throw normalizeApiError(error);
  }
}
