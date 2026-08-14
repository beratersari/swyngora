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

export type PaperOrderType =
  | 'market'
  | 'limit_buy'
  | 'limit_sell'
  | 'stop_loss'
  | 'trailing_stop'
  | 'oco'
  | 'bracket';

export type PlacePortfolioOrderArg = {
  portfolioId?: string;
  exchange?: MarketExchange;
  symbol: string;
  side?: 'buy' | 'sell';
  type?: PaperOrderType;
  quantity: number;
  triggerPrice?: number;
  takeProfitPrice?: number;
  stopLossPrice?: number;
  trailType?: 'percent' | 'offset';
  trailValue?: number;
  lotMethod?: 'fifo' | 'lifo';
  timeInForce?: 'gtc' | 'ioc' | 'fok';
  expiresAt?: string;
  idempotencyKey?: string;
};

export type PlacePortfolioOrderResponse = {
  type?: string;
  trade?: PaperTrade;
  order?: PendingOrder;
  entry?: PendingOrder;
  takeProfit?: PendingOrder;
  stopLoss?: PendingOrder;
  ocoGroupId?: string;
  bracketId?: string;
  portfolio?: PortfolioView;
  note?: string;
};

export type AmendPortfolioOrderArg = {
  id: string;
  portfolioId?: string;
  triggerPrice?: number;
  remainingQuantity?: number;
};

export type AmendPortfolioOrderResponse = {
  order?: PendingOrder;
  portfolio?: PortfolioView;
  note?: string;
};

export type MarginPosition = components['schemas']['MarginPosition'];
export type MarginMode = 'isolated' | 'cross';

export type PlaceMarginOrderArg = {
  portfolioId?: string;
  exchange?: MarketExchange;
  symbol: string;
  side: 'long' | 'short';
  type?: 'market' | 'limit';
  quantity: number;
  leverage: number;
  limitPrice?: number;
  stopLoss?: number;
  takeProfit?: number;
  idempotencyKey?: string;
};

export type PlaceMarginOrderResponse = {
  type?: string;
  position?: MarginPosition;
  order?: {
    id?: string;
    symbol?: string;
    side?: string;
    type?: string;
    quantity?: number;
    leverage?: number;
    limitPrice?: number;
    status?: string;
  };
  note?: string;
};

export type CloseMarginPositionArg = {
  id: string;
  portfolioId?: string;
  quantity?: number;
  idempotencyKey?: string;
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
          takeProfitPrice: body.takeProfitPrice,
          stopLossPrice: body.stopLossPrice,
          trailType: body.trailType,
          trailValue: body.trailValue,
          lotMethod: body.lotMethod,
          timeInForce: body.timeInForce,
          expiresAt: body.expiresAt,
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
    amendPortfolioOrder: build.mutation<AmendPortfolioOrderResponse, AmendPortfolioOrderArg>({
      query: ({ id, portfolioId, triggerPrice, remainingQuantity }) => ({
        url: `/api/v1/portfolio/orders/${encodeURIComponent(id)}`,
        method: 'PATCH',
        params: bookParams(portfolioId),
        body: {
          ...(triggerPrice != null ? { triggerPrice } : {}),
          ...(remainingQuantity != null ? { remainingQuantity } : {}),
        },
      }),
      invalidatesTags: ['Portfolio'],
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
    setMarginMode: build.mutation<
      { marginMode?: MarginMode; clientId?: string },
      { mode: MarginMode; portfolioId?: string }
    >({
      query: (body) => ({
        url: '/api/v1/portfolio/margin/mode',
        method: 'PUT',
        body: { mode: body.mode, portfolioId: body.portfolioId },
      }),
      invalidatesTags: ['Portfolio'],
    }),
    placeMarginOrder: build.mutation<PlaceMarginOrderResponse, PlaceMarginOrderArg>({
      query: (body) => ({
        url: '/api/v1/portfolio/margin/orders',
        method: 'POST',
        body: {
          portfolioId: body.portfolioId,
          exchange: body.exchange ?? 'binance',
          symbol: body.symbol,
          side: body.side,
          type: body.type ?? 'market',
          quantity: body.quantity,
          leverage: body.leverage,
          limitPrice: body.limitPrice,
          stopLoss: body.stopLoss,
          takeProfit: body.takeProfit,
          idempotencyKey: body.idempotencyKey,
        },
        headers: body.idempotencyKey ? { 'Idempotency-Key': body.idempotencyKey } : undefined,
      }),
      invalidatesTags: ['Portfolio'],
    }),
    listMarginPositions: build.query<
      { positions?: MarginPosition[]; count?: number },
      BookArg | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/margin/positions',
        params: bookParams(arg?.portfolioId),
      }),
      providesTags: ['Portfolio'],
    }),
    listMarginOrders: build.query<
      { orders?: PlaceMarginOrderResponse['order'][]; count?: number },
      ({ status?: string } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/margin/orders',
        params: {
          status: arg?.status ?? 'open',
          ...bookParams(arg?.portfolioId),
        },
      }),
      providesTags: ['Portfolio'],
    }),
    cancelMarginOrder: build.mutation<
      { order?: PlaceMarginOrderResponse['order'] },
      { id: string; portfolioId?: string }
    >({
      query: ({ id, portfolioId }) => ({
        url: `/api/v1/portfolio/margin/orders/${encodeURIComponent(id)}`,
        method: 'DELETE',
        params: bookParams(portfolioId),
      }),
      invalidatesTags: ['Portfolio'],
    }),
    closeMarginPosition: build.mutation<
      { position?: MarginPosition; trade?: unknown; note?: string },
      CloseMarginPositionArg
    >({
      query: ({ id, portfolioId, quantity, idempotencyKey }) => ({
        url: `/api/v1/portfolio/margin/positions/${encodeURIComponent(id)}/close`,
        method: 'POST',
        params: bookParams(portfolioId),
        body: {
          ...(quantity != null && quantity > 0 ? { quantity } : {}),
          ...(idempotencyKey ? { idempotencyKey } : {}),
        },
        headers: idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : undefined,
      }),
      invalidatesTags: ['Portfolio'],
    }),
    setMarginBrackets: build.mutation<
      MarginPosition,
      {
        id: string;
        portfolioId?: string;
        stopLoss?: number;
        takeProfit?: number;
        clearStopLoss?: boolean;
        clearTakeProfit?: boolean;
      }
    >({
      query: ({ id, portfolioId, ...body }) => ({
        url: `/api/v1/portfolio/margin/positions/${encodeURIComponent(id)}/brackets`,
        method: 'PUT',
        params: bookParams(portfolioId),
        body,
      }),
      invalidatesTags: ['Portfolio'],
    }),
    repayMarginDebt: build.mutation<
      { position?: MarginPosition; trade?: unknown },
      { id: string; amount: number; portfolioId?: string }
    >({
      query: ({ id, amount, portfolioId }) => ({
        url: `/api/v1/portfolio/margin/positions/${encodeURIComponent(id)}/repay`,
        method: 'POST',
        params: bookParams(portfolioId),
        body: { amount },
      }),
      invalidatesTags: ['Portfolio'],
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
  useAmendPortfolioOrderMutation,
  useCancelPortfolioOrderMutation,
  useListPortfolioTradesQuery,
  useSetMarginModeMutation,
  usePlaceMarginOrderMutation,
  useListMarginPositionsQuery,
  useListMarginOrdersQuery,
  useCancelMarginOrderMutation,
  useCloseMarginPositionMutation,
  useSetMarginBracketsMutation,
  useRepayMarginDebtMutation,
} = portfolioApi;
