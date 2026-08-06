import { baseApi } from '../baseApi';
import type { operations } from '../generated/schema';

/** OpenAPI getHealth 200 body. */
export type HealthResponse = NonNullable<
  operations['getHealth']['responses']['200']['content']['application/json']
>;

export const healthApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getHealth: build.query<HealthResponse, void>({
      query: () => '/health',
      providesTags: ['Health'],
    }),
  }),
});

export const { useGetHealthQuery } = healthApi;
