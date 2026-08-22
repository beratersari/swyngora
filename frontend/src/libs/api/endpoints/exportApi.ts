import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type ExportJob = components['schemas']['ExportJob'];

type ExportListResponse = {
  clientId?: string;
  count?: number;
  exports?: ExportJob[];
};

export const exportApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listExports: build.query<ExportListResponse, void>({
      query: () => '/api/v1/export',
      providesTags: ['ExportJob'],
    }),
    startExport: build.mutation<
      ExportJob,
      { format?: 'json' | 'csv'; sections?: NonNullable<ExportJob['sections']> }
    >({
      query: (body) => ({ url: '/api/v1/export', method: 'POST', body }),
      invalidatesTags: ['ExportJob'],
    }),
    getExport: build.query<ExportJob, { id: string }>({
      query: ({ id }) => `/api/v1/export/${encodeURIComponent(id)}`,
      providesTags: (_r, _e, arg) => [{ type: 'ExportJob', id: arg.id }],
    }),
    cancelExport: build.mutation<ExportJob, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/export/${encodeURIComponent(id)}/cancel`,
        method: 'POST',
      }),
      invalidatesTags: ['ExportJob'],
    }),
  }),
});

export const {
  useListExportsQuery,
  useStartExportMutation,
  useGetExportQuery,
  useCancelExportMutation,
} = exportApi;
