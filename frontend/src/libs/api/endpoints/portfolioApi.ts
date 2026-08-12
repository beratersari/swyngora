import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';
import type { MarketExchange } from './marketApi';

export type PortfolioSummary = components['schemas']['PortfolioSummary'];
export type PortfolioPerformance = components['schemas']['PortfolioPerformance'];
export type PortfolioEquityPoint = components['schemas']['PortfolioEquityPoint'];
export type PortfolioPerformancePeriod = NonNullable<
  NonNullable<components['schemas']['PortfolioPerformance']['period']>
>;
export type PortfolioCashMovement = components['schemas']['PortfolioCashMovement'];
export type PortfolioCashMoveResponse = components['schemas']['PortfolioCashMoveResponse'];
export type PortfolioTransferResponse = components['schemas']['PortfolioTransferResponse'];
export type TaxLot = components['schemas']['TaxLot'];
export type PortfolioShare = components['schemas']['PortfolioShare'];
export type SharedPortfolioSummary = components['schemas']['SharedPortfolioSummary'];
export type PaperTrade = components['schemas']['PaperTrade'];
export type PendingOrder = components['schemas']['PendingOrder'];

/** Spot position row (OpenAPI codegen flattens positions to `Record<string, never>[]`). */
export type SpotPosition = {
  exchange?: string;
  symbol?: string;
  quantity?: number;
  reservedQuantity?: number;
  availableQuantity?: number;
  avgCost?: number;
  markPrice?: number;
  marketValue?: number;
  unrealizedPnL?: number;
  costBasis?: number;
  lots?: TaxLot[];
};

export type PortfolioView = Omit<components['schemas']['PortfolioView'], 'positions'> & {
  positions?: SpotPosition[];
};

export type PlacePortfolioOrderArg = {
  portfolioId?: string;
  exchange?: MarketExchange;
  symbol: string;
  side: 'buy' | 'sell';
  type?: 'market' | 'limit_buy' | 'limit_sell' | 'stop_loss';
  quantity: number;
  triggerPrice?: number;
  lotMethod?: 'fifo' | 'lifo';
  timeInForce?: 'gtc' | 'ioc' | 'fok';
  idempotencyKey?: string;
};

export type PlacePortfolioOrderResponse = {
  type?: string;
  trade?: PaperTrade;
  order?: PendingOrder;
  portfolio?: PortfolioView;
  note?: string;
};

type BookArg = { portfolioId?: string };
type CashMoveArg = { amount: number; note?: string; portfolioId?: string };
type CashMovementListResponse = {
  clientId?: string;
  movements?: PortfolioCashMovement[];
  count?: number;
  total?: number;
  limit?: number;
  offset?: number;
};
type PortfolioListResponse = {
  clientId?: string;
  count?: number;
  portfolios?: PortfolioSummary[];
};
type OrdersListResponse = {
  clientId?: string;
  status?: string;
  count?: number;
  orders?: PendingOrder[];
};
type TradesListResponse = {
  clientId?: string;
  count?: number;
  total?: number;
  limit?: number;
  offset?: number;
  trades?: PaperTrade[];
};

const bookParams = (portfolioId?: string) => (portfolioId ? { portfolioId } : undefined);

export const portfolioApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listPortfolios: build.query<PortfolioListResponse, void>({
      query: () => '/api/v1/portfolios',
      providesTags: ['Portfolio'],
    }),
    getPortfolio: build.query<PortfolioView, BookArg | void>({
      query: (arg) => ({
        url: '/api/v1/portfolio',
        params: bookParams(arg?.portfolioId),
      }),
      providesTags: ['Portfolio'],
    }),
    createPortfolio: build.mutation<PortfolioView, { startingBalance: number; currency?: string; name?: string }>({
      query: (body) => ({ url: '/api/v1/portfolio', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
    renamePortfolio: build.mutation<PortfolioSummary, { id: string; name: string }>({
      query: ({ id, name }) => ({ url: `/api/v1/portfolios/${id}`, method: 'PATCH', body: { name } }),
      invalidatesTags: ['Portfolio'],
    }),
    deletePortfolio: build.mutation<{ deleted?: boolean; id?: string }, string>({
      query: (id) => ({ url: `/api/v1/portfolios/${id}`, method: 'DELETE' }),
      invalidatesTags: ['Portfolio'],
    }),
    getPortfolioPerformance: build.query<
      PortfolioPerformance,
      ({ period?: PortfolioPerformancePeriod } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/performance',
        params: { period: arg?.period ?? '1w', ...bookParams(arg?.portfolioId) },
      }),
      providesTags: ['Portfolio'],
    }),
    listPortfolioCashMovements: build.query<
      CashMovementListResponse,
      ({ limit?: number; offset?: number } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/cash-movements',
        params: { limit: arg?.limit ?? 50, offset: arg?.offset ?? 0, ...bookParams(arg?.portfolioId) },
      }),
      providesTags: ['Portfolio'],
    }),
    depositPortfolioCash: build.mutation<PortfolioCashMoveResponse, CashMoveArg>({
      query: (body) => ({ url: '/api/v1/portfolio/deposits', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
    withdrawPortfolioCash: build.mutation<PortfolioCashMoveResponse, CashMoveArg>({
      query: (body) => ({ url: '/api/v1/portfolio/withdrawals', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
    transferPortfolioCash: build.mutation<
      PortfolioTransferResponse,
      { fromPortfolioId?: string; toPortfolioId: string; amount: number; note?: string }
    >({
      query: (body) => ({ url: '/api/v1/portfolio/transfers', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
    listPortfolioLots: build.query<
      { lots?: TaxLot[]; count?: number },
      ({ exchange?: string; symbol?: string; status?: 'open' | 'closed' | 'all' } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/lots',
        params: {
          exchange: arg?.exchange,
          symbol: arg?.symbol,
          status: arg?.status ?? 'open',
          ...bookParams(arg?.portfolioId),
        },
      }),
      providesTags: ['Portfolio'],
    }),
    listPortfolioShares: build.query<
      { ownerClientId?: string; count?: number; shares?: PortfolioShare[] },
      BookArg | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/shares',
        params: bookParams(arg?.portfolioId),
      }),
      providesTags: ['Portfolio'],
    }),
    listSharedPortfolios: build.query<
      { clientId?: string; count?: number; portfolios?: SharedPortfolioSummary[] },
      void
    >({
      query: () => '/api/v1/portfolios/shared',
      providesTags: ['Portfolio'],
    }),
    sharePortfolio: build.mutation<
      PortfolioShare,
      { granteeClientId: string; role: 'viewer' | 'trader'; portfolioId?: string }
    >({
      query: (body) => ({ url: '/api/v1/portfolio/shares', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
    updatePortfolioShare: build.mutation<
      PortfolioShare,
      { granteeClientId: string; role: 'viewer' | 'trader'; portfolioId?: string }
    >({
      query: (body) => ({ url: '/api/v1/portfolio/shares', method: 'PATCH', body }),
      invalidatesTags: ['Portfolio'],
    }),
    revokePortfolioShare: build.mutation<
      { revoked?: boolean; granteeClientId?: string },
      { granteeClientId: string; portfolioId?: string }
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/shares',
        method: 'DELETE',
        params: { granteeClientId: arg.granteeClientId, ...bookParams(arg.portfolioId) },
      }),
      invalidatesTags: ['Portfolio'],
    }),
    placePortfolioOrder: build.mutation<PlacePortfolioOrderResponse, PlacePortfolioOrderArg>({
      query: (body) => ({
        url: '/api/v1/portfolio/orders',
        method: 'POST',
        body: {
          portfolioId: body.portfolioId,
          exchange: body.exchange ?? 'binance',
          symbol: body.symbol,
          side: body.side,
          type: body.type ?? 'market',
          quantity: body.quantity,
          triggerPrice: body.triggerPrice,
          lotMethod: body.lotMethod,
          timeInForce: body.timeInForce,
          idempotencyKey: body.idempotencyKey,
        },
        headers: body.idempotencyKey ? { 'Idempotency-Key': body.idempotencyKey } : undefined,
      }),
      invalidatesTags: ['Portfolio'],
    }),
    listPortfolioOrders: build.query<
      OrdersListResponse,
      ({ status?: string; limit?: number; offset?: number } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/orders',
        params: {
          status: arg?.status ?? 'open',
          limit: arg?.limit ?? 50,
          offset: arg?.offset ?? 0,
          ...bookParams(arg?.portfolioId),
        },
      }),
      providesTags: ['Portfolio'],
    }),
    cancelPortfolioOrder: build.mutation<
      { order?: PendingOrder; portfolio?: PortfolioView },
      { id: string; portfolioId?: string }
    >({
      query: ({ id, portfolioId }) => ({
        url: `/api/v1/portfolio/orders/${encodeURIComponent(id)}`,
        method: 'DELETE',
        params: bookParams(portfolioId),
      }),
      invalidatesTags: ['Portfolio'],
    }),
    listPortfolioTrades: build.query<
      TradesListResponse,
      ({ limit?: number; offset?: number } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/trades',
        params: {
          limit: arg?.limit ?? 50,
          offset: arg?.offset ?? 0,
          ...bookParams(arg?.portfolioId),
        },
      }),
      providesTags: ['Portfolio'],
    }),
  }),
});

export const {
  useListPortfoliosQuery,
  useGetPortfolioQuery,
  useCreatePortfolioMutation,
  useRenamePortfolioMutation,
  useDeletePortfolioMutation,
  useGetPortfolioPerformanceQuery,
  useListPortfolioCashMovementsQuery,
  useDepositPortfolioCashMutation,
  useWithdrawPortfolioCashMutation,
  useTransferPortfolioCashMutation,
  useListPortfolioLotsQuery,
  useListPortfolioSharesQuery,
  useListSharedPortfoliosQuery,
  useSharePortfolioMutation,
  useUpdatePortfolioShareMutation,
  useRevokePortfolioShareMutation,
  usePlacePortfolioOrderMutation,
  useListPortfolioOrdersQuery,
  useCancelPortfolioOrderMutation,
  useListPortfolioTradesQuery,
} = portfolioApi;
