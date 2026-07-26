import { baseApi } from '../baseApi';

export type HealthResponse = {
  status: string;
  time?: string;
};

export const healthApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getHealth: build.query<HealthResponse, void>({
      query: () => '/health',
      providesTags: ['Health'],
    }),
  }),
});

export const { useGetHealthQuery } = healthApi;
