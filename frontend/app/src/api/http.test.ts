import MockAdapter from 'axios-mock-adapter';
import { afterEach, describe, expect, it } from 'vitest';

import { apiRequest, type ApiError, httpClient } from './http';

const mock = new MockAdapter(httpClient);

afterEach(() => {
  mock.reset();
});

describe('apiRequest', () => {
  it('sends JSON requests to the configured backend through axios', async () => {
    mock.onPost('/test').reply((config) => {
      expect(config.headers?.Authorization).toBe('Bearer access-token');
      expect(config.headers?.['Content-Type']).toContain('application/json');
      expect(JSON.parse(config.data as string)).toEqual({ hello: 'world' });
      return [200, { ok: true }];
    });

    const result = await apiRequest<{ ok: boolean }, { hello: string }>('/test', {
      method: 'POST',
      accessToken: 'access-token',
      data: { hello: 'world' },
    });

    expect(result).toEqual({ ok: true });
  });

  it('throws ApiError with backend message and code', async () => {
    mock.onGet('/login').reply(403, { message: 'banned', code: 'account_banned' });

    await expect(apiRequest('/login')).rejects.toMatchObject({
      name: 'ApiError',
      message: 'banned',
      status: 403,
      code: 'account_banned',
    } satisfies Partial<ApiError>);
  });
});
