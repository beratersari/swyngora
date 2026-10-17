import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, message } from 'antd';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import {
  PaperMarginForm,
  type PaperMarginFormProps,
  type PaperMarginFormValues,
} from '@/components/organisms/PaperMarginForm';
import {
  PaperTradeForm,
  type PaperTradeFormProps,
  type PaperTradeFormValues,
} from '@/components/organisms/PaperTradeForm';
import { PortfolioBookSelect } from '@/components/organisms/PortfolioBookSelect';
import { PortfolioCashPanel } from '@/components/organisms/PortfolioCashPanel';
import { PortfolioEquityChart } from '@/components/organisms/PortfolioEquityChart';
import { PortfolioRiskLimits } from '@/components/organisms/PortfolioRiskLimits';
import { PortfolioMarginPositionsTable } from '@/components/organisms/PortfolioMarginPositionsTable';
import { PortfolioOrdersTable } from '@/components/organisms/PortfolioOrdersTable';
import { SpreadOrderList } from '@/components/organisms/SpreadOrderList';
import { PortfolioPositionsTable } from '@/components/organisms/PortfolioPositionsTable';
import { PortfolioSituationLeave } from '@/components/organisms/PortfolioSituationLeave';
import { PortfolioSituationRecord } from '@/components/organisms/PortfolioSituationRecord';
import { PortfolioSummaryStrip } from '@/components/organisms/PortfolioSummaryStrip';
import { PortfolioStressPanel } from '@/components/organisms/PortfolioStressPanel';
import { PortfolioTradesTable } from '@/components/organisms/PortfolioTradesTable';
import { RebalanceResultsTable } from '@/components/organisms/RebalanceResultsTable';
import {
  getRtkErrorStatus,
  rtkErrorMessage,
  useAmendPortfolioOrderMutation,
  useCancelPortfolioOrderMutation,
  useCancelSpreadOrderMutation,
  useListSpreadOrdersQuery,
  useCloseMarginPositionMutation,
  useCreatePortfolioMutation,
  useDepositPortfolioCashMutation,
  useListTransferAttemptsQuery,
  useGetPortfolioPerformanceQuery,
  useGetPortfolioQuery,
  useGetPortfolioSituationLeaveQuery,
  useGetPortfolioSituationRecordQuery,
  useListMarginPositionsQuery,
  useListPortfolioCashMovementsQuery,
  useListPortfolioOrdersQuery,
  useContinuePortfolioRebalanceMutation,
  useGetPortfolioRebalanceQuery,
  useListPortfolioRebalancesQuery,
  useListPortfolioTradesQuery,
  useListPortfoliosQuery,
  usePlaceMarginOrderMutation,
  usePlacePortfolioOrderMutation,
  useSetMarginBracketsMutation,
  useSetMarginModeMutation,
  useStressPortfolioMutation,
  useTransferPortfolioCashMutation,
  useWithdrawPortfolioCashMutation,
  type PortfolioPerformancePeriod,
} from '@/libs/api';
import { useDocumentVisible, useMediaQuery, useOrderPreview } from '@/libs/hooks';
import type { MarginOrderPreviewDraft, SpotOrderPreviewDraft } from '@/libs/hooks/orderPreview';
import { usePortfolioSubscription } from '@/libs/realtime';
import {
  formatDateTime,
  formatPrice,
  newPaperIdempotencyKey,
  rtkCurrent,
  rtkCurrentPending,
} from '@/libs/utils';
import { mediaQueries } from '@/styles/tokens';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import { DeskTabs, PageStack, PanelCard, Section, Split } from './PortfolioPage.styles';

const BOOK_KEY = 'swyngora.portfolioBookId';

/**
 * Paper trading desk: books, cash, market orders, positions, pending orders, trades, equity.
 * Informational only — not real money.
 */
export function PortfolioPage() {
  const { t } = useTranslation(['portfolio', 'common']);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();
  const isPhone = useMediaQuery(mediaQueries.phone);
  const chartHeight = isPhone ? 180 : 220;
  const booksQuery = useListPortfoliosQuery(undefined, { refetchOnFocus: true });
  const books = booksQuery.data?.portfolios ?? [];

  const paramBook = searchParams.get('book')?.trim() || '';
  const [selectedId, setSelectedId] = useState(() => {
    if (paramBook) return paramBook;
    try {
      return localStorage.getItem(BOOK_KEY) ?? '';
    } catch {
      return '';
    }
  });

  useEffect(() => {
    if (paramBook && paramBook !== selectedId) setSelectedId(paramBook);
  }, [paramBook, selectedId]);

  useEffect(() => {
    if (selectedId) {
      try {
        localStorage.setItem(BOOK_KEY, selectedId);
      } catch {
        /* ignore */
      }
      return;
    }
    if (books.length === 1 && books[0]?.id) {
      setSelectedId(books[0].id);
    }
  }, [books, selectedId]);

  const bookId = selectedId || undefined;
  const needBook = books.length > 1 && !bookId;
  usePortfolioSubscription(bookId, visible && Boolean(bookId));

  const portfolioQuery = useGetPortfolioQuery(bookId ? { portfolioId: bookId } : undefined, {
    skip: !bookId && books.length !== 1,
    pollingInterval: visible ? 15_000 : 0,
    refetchOnFocus: true,
  });
  const [period, setPeriod] = useState<PortfolioPerformancePeriod>('1w');
  const perfQuery = useGetPortfolioPerformanceQuery(
    { period, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const spreadQuery = useListSpreadOrdersQuery(
    { status: 'open', portfolioId: bookId },
    {
      skip: !bookId && books.length !== 1,
      pollingInterval: visible ? 10_000 : 0,
      refetchOnFocus: true,
    },
  );
  const ordersQuery = useListPortfolioOrdersQuery(
    { status: 'open', portfolioId: bookId },
    {
      skip: !bookId && books.length !== 1,
      pollingInterval: visible ? 10_000 : 0,
      refetchOnFocus: true,
    },
  );
  const recordQuery = useGetPortfolioSituationRecordQuery(
    { portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const leaveQuery = useGetPortfolioSituationLeaveQuery(
    { portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const tradesQuery = useListPortfolioTradesQuery(
    { limit: 50, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const [expandedRebalance, setExpandedRebalance] = useState<string | undefined>();
  const rebalancesQuery = useListPortfolioRebalancesQuery(
    { limit: 20, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const rebalanceDetail = useGetPortfolioRebalanceQuery(
    { id: expandedRebalance ?? '', portfolioId: bookId },
    { skip: !expandedRebalance },
  );
  const cashQuery = useListPortfolioCashMovementsQuery(
    { limit: 30, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const attemptsQuery = useListTransferAttemptsQuery(
    { limit: 20, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const marginPosQuery = useListMarginPositionsQuery(
    { portfolioId: bookId },
    {
      skip: !bookId && books.length !== 1,
      pollingInterval: visible ? 12_000 : 0,
      refetchOnFocus: true,
    },
  );

  const [createBook, createState] = useCreatePortfolioMutation();
  const [placeOrder, placeState] = usePlacePortfolioOrderMutation();
  const [amendOrder, amendState] = useAmendPortfolioOrderMutation();
  const [cancelOrder, cancelState] = useCancelPortfolioOrderMutation();
  const [cancelSpread, cancelSpreadState] = useCancelSpreadOrderMutation();
  const [deposit, depositState] = useDepositPortfolioCashMutation();
  const [withdraw, withdrawState] = useWithdrawPortfolioCashMutation();
  const [transfer, transferState] = useTransferPortfolioCashMutation();
  const transferKeyRef = useRef<string | null>(null);
  const [setMarginMode, marginModeState] = useSetMarginModeMutation();
  const [stressBook, stressState] = useStressPortfolioMutation();
  const [placeMargin, placeMarginState] = usePlaceMarginOrderMutation();
  const previewOn = !needBook && (books.length === 1 || Boolean(bookId));
  const [closeMargin, closeMarginState] = useCloseMarginPositionMutation();
  const [setMarginBrackets, marginBracketsState] = useSetMarginBracketsMutation();
  const [continueRebalance, continueRebalanceState] = useContinuePortfolioRebalanceMutation();

  const view = rtkCurrent(portfolioQuery);
  const positions = view?.positions ?? [];
  const marginPositions = rtkCurrent(marginPosQuery)?.positions ?? view?.marginPositions ?? [];
  const currency = view?.currency ?? 'USDT';

  const selectBook = (id: string) => {
    setSelectedId(id);
    const next = new URLSearchParams(searchParams);
    next.set('book', id);
    setSearchParams(next, { replace: true });
  };

  const openMarket = (exchange: string, symbol: string) => {
    navigate(`/markets/${encodeURIComponent(exchange)}/${encodeURIComponent(symbol)}`);
  };

  const onTrade = async (values: PaperTradeFormValues) => {
    if (needBook) return;
    const res = await placeOrder({
      portfolioId: bookId,
      exchange: values.exchange,
      symbol: values.symbol,
      side: values.side,
      type: values.orderType,
      quantity: values.quantity,
      triggerPrice: values.triggerPrice,
      takeProfitPrice: values.takeProfitPrice,
      stopLossPrice: values.stopLossPrice,
      trailType: values.trailType,
      trailValue: values.trailValue,
      lotMethod: values.lotMethod,
      timeInForce: values.timeInForce,
      idempotencyKey: newPaperIdempotencyKey('web'),
      maxSlippagePct: values.maxSlippagePct,
      spread: values.spread,
      reason: values.reason,
    }).unwrap();
    const isSpread = Boolean(res.spread);
    const isPending = Boolean(res.order || res.entry || res.ocoGroupId || res.bracketId);
    void message.success(
      isSpread
        ? t('portfolio:trade.successSpread', { symbol: values.symbol })
        : isPending
        ? t('portfolio:trade.successPending', { symbol: values.symbol })
        : values.side === 'buy'
          ? t('portfolio:trade.successBuy', { qty: values.quantity, symbol: values.symbol })
          : t('portfolio:trade.successSell', { qty: values.quantity, symbol: values.symbol }),
    );
  };

  const onMargin = async (values: PaperMarginFormValues) => {
    if (needBook) return;
    await placeMargin({
      portfolioId: bookId,
      exchange: values.exchange,
      symbol: values.symbol,
      side: values.side,
      type: values.orderType,
      quantity: values.quantity,
      leverage: values.leverage,
      limitPrice: values.limitPrice,
      stopLoss: values.stopLoss,
      takeProfit: values.takeProfit,
      idempotencyKey: `web-m-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`,
    }).unwrap();
    void message.success(
      t('portfolio:margin.successOpen', { symbol: values.symbol, lev: values.leverage }),
    );
  };

  const cashColumns = useMemo(
    () => [
      {
        title: t('portfolio:movements.kind'),
        dataIndex: 'kind',
        key: 'kind',
      },
      {
        title: t('portfolio:movements.amount'),
        dataIndex: 'amount',
        key: 'amount',
        align: 'right' as const,
        render: (v: number | undefined) => formatPrice(v),
      },
      {
        title: t('portfolio:movements.after'),
        dataIndex: 'cashAfter',
        key: 'after',
        align: 'right' as const,
        render: (v: number | undefined) => formatPrice(v),
      },
      {
        title: t('portfolio:movements.time'),
        dataIndex: 'createdAt',
        key: 't',
        render: (v: string | undefined) => formatDateTime(v),
      },
    ],
    [t],
  );

  const attemptColumns = useMemo(
    () => [
      {
        title: t('portfolio:transfers.attempt'),
        dataIndex: 'attempt',
        key: 'attempt',
      },
      {
        title: t('portfolio:transfers.status'),
        dataIndex: 'status',
        key: 'status',
        render: (v: string | undefined) => {
          if (v === 'completed') return t('portfolio:transfers.completed');
          if (v === 'replayed') return t('portfolio:transfers.replayed');
          if (v === 'conflict') return t('portfolio:transfers.conflict');
          return v ?? '—';
        },
      },
      {
        title: t('portfolio:transfers.from'),
        dataIndex: 'fromPortfolioName',
        key: 'from',
      },
      {
        title: t('portfolio:transfers.to'),
        dataIndex: 'toPortfolioName',
        key: 'to',
      },
      {
        title: t('portfolio:movements.amount'),
        dataIndex: 'amount',
        key: 'amount',
        align: 'right' as const,
        render: (v: number | undefined) => formatPrice(v),
      },
      {
        title: t('portfolio:movements.time'),
        dataIndex: 'createdAt',
        key: 't',
        render: (v: string | undefined) => formatDateTime(v),
      },
    ],
    [t],
  );

  return (
    <PageStack>
      <PageHeader title={t('portfolio:title')} />
      <Text variant="caption" color="secondary">
        {t('portfolio:disclaimer')}
      </Text>

      <Section>
        <PortfolioBookSelect
          books={books}
          selectedId={selectedId || view?.id}
          loading={booksQuery.isLoading}
          creating={createState.isLoading}
          onSelect={selectBook}
          onCreate={async ({ name, startingBalance }) => {
            const created = await createBook({ name, startingBalance }).unwrap();
            void message.success(t('portfolio:books.createSuccess'));
            if (created.id) selectBook(created.id);
            void booksQuery.refetch();
          }}
        />
      </Section>

      {booksQuery.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('portfolio:loadFailed')}
          description={rtkErrorMessage(booksQuery.error, { resource: t('portfolio:resource') })}
          action={
            <Button size="small" onClick={() => void booksQuery.refetch()}>
              {t('common:actions.retry')}
            </Button>
          }
        />
      ) : null}

      {books.length === 0 && !booksQuery.isLoading ? (
        <Alert type="info" showIcon message={t('portfolio:books.empty')} />
      ) : null}

      {portfolioQuery.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('portfolio:loadFailed')}
          description={rtkErrorMessage(portfolioQuery.error, { resource: t('portfolio:resource') })}
        />
      ) : null}

      {needBook ? <Alert type="info" showIcon message={t('portfolio:books.select')} /> : null}

      {(books.length > 0 || view) && (
        <>
          <PortfolioSummaryStrip
            view={view}
            isLoading={rtkCurrentPending(portfolioQuery)}
            currency={currency}
          />

          <Split>
            <PanelCard>
              <SpotOrderTicket
                portfolioId={bookId}
                previewEnabled={previewOn}
                visible={visible}
                isSubmitting={placeState.isLoading}
                disabled={needBook}
                submitError={placeState.isError ? placeState.error : undefined}
                onSubmit={onTrade}
              />
            </PanelCard>
            <PanelCard>
              <PortfolioCashPanel
                isDepositing={depositState.isLoading}
                isWithdrawing={withdrawState.isLoading}
                isTransferring={transferState.isLoading}
                disabled={needBook}
                depositError={depositState.isError ? depositState.error : undefined}
                withdrawError={withdrawState.isError ? withdrawState.error : undefined}
                transferError={transferState.isError ? transferState.error : undefined}
                transferBooks={books
                  .filter((b) => b.id && b.id !== bookId)
                  .map((b) => ({ id: b.id as string, name: b.name }))}
                onDeposit={async (amount, note) => {
                  if (needBook) return;
                  await deposit({ amount, note, portfolioId: bookId }).unwrap();
                  void message.success(t('portfolio:cash.depositSuccess'));
                }}
                onWithdraw={async (amount, note) => {
                  if (needBook) return;
                  await withdraw({ amount, note, portfolioId: bookId }).unwrap();
                  void message.success(t('portfolio:cash.withdrawSuccess'));
                }}
                onTransfer={async (toPortfolioId, amount, note) => {
                  if (needBook || !bookId) return;
                  if (!transferKeyRef.current) {
                    transferKeyRef.current = newPaperIdempotencyKey('xfer');
                  }
                  try {
                    const got = await transfer({
                      fromPortfolioId: bookId,
                      toPortfolioId,
                      amount,
                      note,
                      idempotencyKey: transferKeyRef.current,
                    }).unwrap();
                    transferKeyRef.current = null;
                    void message.success(
                      got.replayed
                        ? t('portfolio:cash.transferReplayed')
                        : t('portfolio:cash.transferSuccess'),
                    );
                  } catch (err) {
                    const status = getRtkErrorStatus(err);
                    if (status && status >= 400 && status < 500 && status !== 408) {
                      transferKeyRef.current = null;
                    }
                    throw err;
                  }
                }}
              />
            </PanelCard>
          </Split>

          <Section>
            <PanelCard>
              <MarginOrderTicket
                portfolioId={bookId}
                previewEnabled={previewOn}
                visible={visible}
                marginMode={view?.marginMode}
                modeLoading={marginModeState.isLoading}
                isSubmitting={placeMarginState.isLoading}
                disabled={needBook}
                submitError={placeMarginState.isError ? placeMarginState.error : undefined}
                onModeChange={async (mode) => {
                  await setMarginMode({ mode, portfolioId: bookId }).unwrap();
                  void message.success(t('portfolio:margin.modeSuccess', { mode }));
                }}
                onSubmit={onMargin}
              />
            </PanelCard>
          </Section>

          <Section>
            <PanelCard>
              <PortfolioRiskLimits portfolioId={bookId} disabled={needBook} />
            </PanelCard>
          </Section>

          <Section>
            <PanelCard>
              <PortfolioStressPanel
                disabled={needBook}
                isLoading={stressState.isLoading}
                report={stressState.data}
                errorMessage={
                  stressState.isError
                    ? rtkErrorMessage(stressState.error, { resource: t('portfolio:resource') })
                    : null
                }
                onPreview={(moves) => {
                  void stressBook({ portfolioId: bookId, moves });
                }}
              />
            </PanelCard>
          </Section>

          <Section>
            <PortfolioEquityChart
              points={rtkCurrent(perfQuery)?.points}
              startEquity={rtkCurrent(perfQuery)?.startEquity}
              startAt={rtkCurrent(perfQuery)?.startAt}
              maxDrawdown={rtkCurrent(perfQuery)?.maxDrawdown}
              maxDrawdownPct={rtkCurrent(perfQuery)?.maxDrawdownPct}
              drawdownFrom={rtkCurrent(perfQuery)?.drawdownFrom}
              drawdownTo={rtkCurrent(perfQuery)?.drawdownTo}
              changeAmount={rtkCurrent(perfQuery)?.changeAmount}
              changePct={rtkCurrent(perfQuery)?.changePct}
              currency={currency}
              period={period}
              onPeriodChange={setPeriod}
              isLoading={rtkCurrentPending(perfQuery) || perfQuery.isFetching}
              isError={perfQuery.isError}
              height={chartHeight}
            />
          </Section>

          {bookId || books.length === 1 ? (
            <Section>
              <PanelCard>
                <PortfolioSituationRecord
                  record={rtkCurrent(recordQuery)}
                  loading={rtkCurrentPending(recordQuery)}
                  failed={recordQuery.isError}
                />
              </PanelCard>
            </Section>
          ) : null}

          {bookId || books.length === 1 ? (
            <Section>
              <PanelCard>
                <PortfolioSituationLeave
                  record={rtkCurrent(leaveQuery)}
                  loading={rtkCurrentPending(leaveQuery)}
                  failed={leaveQuery.isError}
                />
              </PanelCard>
            </Section>
          ) : null}

          <DeskTabs
            size={isPhone ? 'small' : 'middle'}
            items={[
              {
                key: 'positions',
                label: t('portfolio:positions.title'),
                children: (
                  <PortfolioPositionsTable
                    items={positions}
                    loading={rtkCurrentPending(portfolioQuery)}
                    portfolioId={bookId}
                    onOpen={openMarket}
                  />
                ),
              },
              {
                key: 'orders',
                label: t('portfolio:orders.title'),
                children: (
                  <>
                  <SpreadOrderList
                    items={(rtkCurrent(spreadQuery)?.orders ?? []).flatMap((order) =>
                      order.id
                        ? [
                            {
                              id: order.id,
                              symbol: order.symbol ?? '',
                              side: order.side ?? '',
                              quantity: order.quantity ?? 0,
                              filledQuantity: order.filledQuantity ?? 0,
                              sliceCount: order.sliceCount ?? 0,
                              slicesDone: order.slicesDone ?? 0,
                            },
                          ]
                        : [],
                    )}
                    canceling={cancelSpreadState.isLoading}
                    onCancel={(id) => {
                      void cancelSpread({ id, portfolioId: bookId })
                        .unwrap()
                        .then(() => message.success(t('portfolio:orders.cancelSuccess')))
                        .catch((err: unknown) => {
                          void message.error(
                            rtkErrorMessage(err, { resource: t('portfolio:orders.spreadTitle') }),
                          );
                        });
                    }}
                  />
                  <PortfolioOrdersTable
                    items={rtkCurrent(ordersQuery)?.orders ?? []}
                    loading={rtkCurrentPending(ordersQuery)}
                    cancelLoading={cancelState.isLoading}
                    amendLoading={amendState.isLoading}
                    onOpen={openMarket}
                    onAmend={async (values) => {
                      await amendOrder({
                        id: values.id,
                        portfolioId: bookId,
                        triggerPrice: values.triggerPrice,
                        remainingQuantity: values.remainingQuantity,
                      }).unwrap();
                      void message.success(t('portfolio:orders.amendSuccess'));
                    }}
                    onCancel={(id) => {
                      void cancelOrder({ id, portfolioId: bookId })
                        .unwrap()
                        .then(() => message.success(t('portfolio:orders.cancelSuccess')))
                        .catch((err: unknown) => {
                          void message.error(
                            rtkErrorMessage(err, { resource: t('portfolio:orders.title') }),
                          );
                        });
                    }}
                  />
                  </>
                ),
              },
              {
                key: 'margin',
                label: t('portfolio:margin.positionsTitle'),
                children: (
                  <PortfolioMarginPositionsTable
                    items={marginPositions}
                    loading={rtkCurrentPending(marginPosQuery) || rtkCurrentPending(portfolioQuery)}
                    closeLoading={closeMarginState.isLoading}
                    bracketsLoading={marginBracketsState.isLoading}
                    onOpen={openMarket}
                    onClose={async (id, quantity) => {
                      try {
                        await closeMargin({
                          id,
                          portfolioId: bookId,
                          quantity,
                          idempotencyKey: `web-mc-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
                        }).unwrap();
                        void message.success(
                          quantity != null && quantity > 0
                            ? t('portfolio:margin.closePartialSuccess', {
                                qty: formatPrice(quantity),
                              })
                            : t('portfolio:margin.closeSuccess'),
                        );
                      } catch (err: unknown) {
                        void message.error(
                          rtkErrorMessage(err, { resource: t('portfolio:margin.positionsTitle') }),
                        );
                        throw err;
                      }
                    }}
                    onBrackets={async (values) => {
                      try {
                        await setMarginBrackets({
                          id: values.id,
                          portfolioId: bookId,
                          stopLoss: values.stopLoss,
                          takeProfit: values.takeProfit,
                          clearStopLoss: values.clearStopLoss,
                          clearTakeProfit: values.clearTakeProfit,
                        }).unwrap();
                        void message.success(t('portfolio:margin.bracketsSuccess'));
                      } catch (err: unknown) {
                        void message.error(
                          rtkErrorMessage(err, { resource: t('portfolio:margin.positionsTitle') }),
                        );
                        throw err;
                      }
                    }}
                  />
                ),
              },
              {
                key: 'rebalances',
                label: t('portfolio:rebalance.title'),
                children: (
                  <RebalanceResultsTable
                    results={rtkCurrent(rebalancesQuery)?.rebalances ?? []}
                    isLoading={rtkCurrentPending(rebalancesQuery)}
                    expandedId={expandedRebalance}
                    onExpand={setExpandedRebalance}
                    detail={rebalanceDetail.data}
                    detailLoading={rebalanceDetail.isFetching}
                    continueLoading={continueRebalanceState.isLoading}
                    onContinue={async (id) => {
                      try {
                        await continueRebalance({ id, portfolioId: bookId }).unwrap();
                        void message.success(t('portfolio:rebalance.continueSuccess'));
                      } catch (err) {
                        void message.error(
                          rtkErrorMessage(err, { resource: t('portfolio:rebalance.title') }),
                        );
                      }
                    }}
                  />
                ),
              },
              {
                key: 'trades',
                label: t('portfolio:trades.title'),
                children: (
                  <PortfolioTradesTable
                    items={rtkCurrent(tradesQuery)?.trades ?? []}
                    loading={rtkCurrentPending(tradesQuery)}
                    onOpen={openMarket}
                  />
                ),
              },
              {
                key: 'cash',
                label: t('portfolio:movements.title'),
                children: (
                  <>
                    {!rtkCurrentPending(cashQuery) &&
                    (rtkCurrent(cashQuery)?.movements?.length ?? 0) === 0 ? (
                      <Text variant="body" color="secondary">
                        {t('portfolio:movements.empty')}
                      </Text>
                    ) : (
                      <DataTableCard>
                        <DataTable
                          rowKey={(r) => (r as { id?: string }).id ?? Math.random().toString()}
                          size="small"
                          loading={rtkCurrentPending(cashQuery)}
                          pagination={{ pageSize: 8, hideOnSinglePage: true, simple: isPhone }}
                          columns={cashColumns}
                          dataSource={rtkCurrent(cashQuery)?.movements ?? []}
                          scroll={{ x: 480 }}
                        />
                      </DataTableCard>
                    )}
                    {(rtkCurrent(attemptsQuery)?.attempts?.length ?? 0) > 0 ? (
                      <>
                        <Text variant="h4" color="primary">
                          {t('portfolio:transfers.requests')}
                        </Text>
                        <DataTableCard>
                          <DataTable
                            rowKey={(r) => (r as { id?: string }).id ?? Math.random().toString()}
                            size="small"
                            loading={rtkCurrentPending(attemptsQuery)}
                            pagination={{ pageSize: 8, hideOnSinglePage: true, simple: isPhone }}
                            columns={attemptColumns}
                            dataSource={rtkCurrent(attemptsQuery)?.attempts ?? []}
                            scroll={{ x: 560 }}
                          />
                        </DataTableCard>
                      </>
                    ) : null}
                  </>
                ),
              },
            ]}
          />
        </>
      )}
    </PageStack>
  );
}

type TicketPreviewProps = {
  portfolioId?: string;
  previewEnabled: boolean;
  visible: boolean;
};

/**
 * Owns the draft so each keystroke reprices this ticket only.
 * The rest of the desk does not render again until the book changes.
 */
function SpotOrderTicket({
  portfolioId,
  previewEnabled,
  visible,
  ...form
}: Omit<PaperTradeFormProps, 'preview' | 'previewLoading' | 'previewFailed' | 'onDraftChange'> &
  TicketPreviewProps) {
  const [draft, setDraft] = useState<SpotOrderPreviewDraft | null>(null);
  const priced = useOrderPreview(draft, { portfolioId, enabled: previewEnabled, visible });
  return (
    <PaperTradeForm
      {...form}
      preview={priced.preview}
      previewLoading={priced.loading}
      previewFailed={priced.failed}
      onDraftChange={setDraft}
    />
  );
}

function MarginOrderTicket({
  portfolioId,
  previewEnabled,
  visible,
  ...form
}: Omit<PaperMarginFormProps, 'preview' | 'previewLoading' | 'previewFailed' | 'onDraftChange'> &
  TicketPreviewProps) {
  const [draft, setDraft] = useState<MarginOrderPreviewDraft | null>(null);
  const priced = useOrderPreview(draft, { portfolioId, enabled: previewEnabled, visible });
  return (
    <PaperMarginForm
      {...form}
      preview={priced.preview}
      previewLoading={priced.loading}
      previewFailed={priced.failed}
      onDraftChange={setDraft}
    />
  );
}
