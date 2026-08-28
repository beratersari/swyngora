import { baseApi } from '../baseApi';
import type {
  CreateScannerRuleArg,
  ScannerBacktest,
  ScannerBacktestListResponse,
  ScannerBacktestSignalListResponse,
  ScannerResultListResponse,
  ScannerRule,
  ScannerRuleListResponse,
  StartScannerBacktestArg,
} from './scannerApi.types';

export type {
  CreateScannerRuleArg,
  ScannerBacktest,
  ScannerBacktestSignal,
  ScannerBacktestStatus,
  ScannerCondition,
  ScannerMaDirection,
  ScannerMatchMode,
  ScannerResult,
  ScannerRsiCondition,
  ScannerRule,
  ScannerRuleType,
  StartScannerBacktestArg,
} from './scannerApi.types';

export const scannerApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listScannerRules: build.query<ScannerRuleListResponse, void>({
      query: () => '/api/v1/scanner/rules',
      providesTags: ['ScannerRule'],
    }),
    createScannerRule: build.mutation<ScannerRule, CreateScannerRuleArg>({
      query: (body) => ({
        url: '/api/v1/scanner/rules',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['ScannerRule'],
    }),
    deleteScannerRule: build.mutation<{ deleted?: boolean; id?: string }, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/scanner/rules/${encodeURIComponent(id)}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['ScannerRule', 'ScannerResult'],
    }),
    listScannerResults: build.query<ScannerResultListResponse, { limit?: number; offset?: number } | void>({
      query: (arg) => ({
        url: '/api/v1/scanner/results',
        params: {
          limit: arg?.limit ?? 100,
          offset: arg?.offset ?? 0,
        },
      }),
      providesTags: ['ScannerResult'],
    }),
    listScannerBacktests: build.query<
      ScannerBacktestListResponse,
      { limit?: number; offset?: number } | void
    >({
      query: (arg) => ({
        url: '/api/v1/scanner/backtests',
        params: {
          limit: arg?.limit ?? 50,
          offset: arg?.offset ?? 0,
        },
      }),
      providesTags: ['ScannerBacktest'],
    }),
    startScannerBacktest: build.mutation<ScannerBacktest, StartScannerBacktestArg>({
      query: (body) => ({
        url: '/api/v1/scanner/backtests',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['ScannerBacktest'],
    }),
    getScannerBacktest: build.query<ScannerBacktest, { id: string }>({
      query: ({ id }) => `/api/v1/scanner/backtests/${encodeURIComponent(id)}`,
      providesTags: (_r, _e, arg) => [{ type: 'ScannerBacktest', id: arg.id }],
    }),
    cancelScannerBacktest: build.mutation<ScannerBacktest, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/scanner/backtests/${encodeURIComponent(id)}/cancel`,
        method: 'POST',
      }),
      invalidatesTags: ['ScannerBacktest'],
    }),
    listScannerBacktestSignals: build.query<
      ScannerBacktestSignalListResponse,
      { id: string; limit?: number; offset?: number }
    >({
      query: ({ id, limit = 100, offset = 0 }) => ({
        url: `/api/v1/scanner/backtests/${encodeURIComponent(id)}/signals`,
        params: { limit, offset },
      }),
    }),
  }),
});

export const {
  useListScannerRulesQuery,
  useCreateScannerRuleMutation,
  useDeleteScannerRuleMutation,
  useListScannerResultsQuery,
  useListScannerBacktestsQuery,
  useStartScannerBacktestMutation,
  useGetScannerBacktestQuery,
  useCancelScannerBacktestMutation,
  useListScannerBacktestSignalsQuery,
} = scannerApi;
