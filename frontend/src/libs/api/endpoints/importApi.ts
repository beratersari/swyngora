import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type ImportJob = components['schemas']['ImportJob'];
export type ImportSectionCounts = components['schemas']['ImportSectionCounts'];

type ImportListResponse = {
  clientId?: string;
  count?: number;
  imports?: ImportJob[];
};

export const importApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listImports: build.query<ImportListResponse, void>({
      query: () => '/api/v1/import',
      providesTags: ['ImportJob'],
    }),
    previewImport: build.mutation<ImportJob, { file: File; format?: 'json' | 'csv' }>({
      query: ({ file, format }) => {
        const body = new FormData();
        body.append('file', file);
        if (format) body.append('format', format);
        return { url: '/api/v1/import/preview', method: 'POST', body };
      },
      invalidatesTags: ['ImportJob'],
    }),
    getImport: build.query<ImportJob, { id: string }>({
      query: ({ id }) => `/api/v1/import/${encodeURIComponent(id)}`,
      providesTags: (_r, _e, arg) => [{ type: 'ImportJob', id: arg.id }],
    }),
    confirmImport: build.mutation<ImportJob, { id: string; mode: 'merge' | 'replace' }>({
      query: ({ id, mode }) => ({
        url: `/api/v1/import/${encodeURIComponent(id)}/confirm`,
        method: 'POST',
        body: { mode },
      }),
      invalidatesTags: ['ImportJob', 'CustomSignalProfile'],
    }),
    cancelImport: build.mutation<ImportJob, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/import/${encodeURIComponent(id)}/cancel`,
        method: 'POST',
      }),
      invalidatesTags: ['ImportJob'],
    }),
  }),
});

export const {
  useListImportsQuery,
  usePreviewImportMutation,
  useGetImportQuery,
  useConfirmImportMutation,
  useCancelImportMutation,
} = importApi;
