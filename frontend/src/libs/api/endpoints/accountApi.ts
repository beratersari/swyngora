import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type AccountAPIKey = components['schemas']['AccountAPIKey'];
export type AccountAPIKeyCreated = components['schemas']['AccountAPIKeyCreated'];

type APIKeyListResponse = {
  clientId?: string;
  keys?: AccountAPIKey[];
  count?: number;
};

export const accountApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listAccountAPIKeys: build.query<APIKeyListResponse, void>({
      query: () => '/api/v1/account/api-keys',
      providesTags: ['AccountAPIKey'],
    }),
    createAccountAPIKey: build.mutation<
      AccountAPIKeyCreated,
      { name: string; permission?: 'read' | 'trade' }
    >({
      query: (body) => ({ url: '/api/v1/account/api-keys', method: 'POST', body }),
      invalidatesTags: ['AccountAPIKey'],
    }),
    revokeAccountAPIKey: build.mutation<AccountAPIKey, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/account/api-keys/${encodeURIComponent(id)}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['AccountAPIKey'],
    }),
  }),
});

export const {
  useListAccountAPIKeysQuery,
  useCreateAccountAPIKeyMutation,
  useRevokeAccountAPIKeyMutation,
} = accountApi;
