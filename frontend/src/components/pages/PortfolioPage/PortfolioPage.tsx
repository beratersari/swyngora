import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, message, Tabs } from 'antd';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { PaperTradeForm, type PaperTradeFormValues } from '@/components/organisms/PaperTradeForm';
import { PortfolioBookSelect } from '@/components/organisms/PortfolioBookSelect';
import { PortfolioCashPanel } from '@/components/organisms/PortfolioCashPanel';
import { PortfolioEquityChart } from '@/components/organisms/PortfolioEquityChart';
import { PortfolioOrdersTable } from '@/components/organisms/PortfolioOrdersTable';
import { PortfolioPositionsTable } from '@/components/organisms/PortfolioPositionsTable';
import { PortfolioSummaryStrip } from '@/components/organisms/PortfolioSummaryStrip';
import { PortfolioTradesTable } from '@/components/organisms/PortfolioTradesTable';
import {
  rtkErrorMessage,
  useCancelPortfolioOrderMutation,
  useCreatePortfolioMutation,
  useDepositPortfolioCashMutation,
  useGetPortfolioPerformanceQuery,
  useGetPortfolioQuery,
  useListPortfolioCashMovementsQuery,
  useListPortfolioOrdersQuery,
  useListPortfolioTradesQuery,
  useListPortfoliosQuery,
  usePlacePortfolioOrderMutation,
  useWithdrawPortfolioCashMutation,
  type PortfolioPerformancePeriod,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { usePortfolioSubscription } from '@/libs/realtime';
import { formatDateTime, formatPrice } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import { PageStack, PanelCard, Section, Split } from './PortfolioPage.styles';

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
  usePortfolioSubscription(bookId, visible && Boolean(bookId));

  const portfolioQuery = useGetPortfolioQuery(
    bookId ? { portfolioId: bookId } : undefined,
    { skip: books.length === 0 && !bookId, pollingInterval: visible ? 15_000 : 0, refetchOnFocus: true },
  );
  const [period, setPeriod] = useState<PortfolioPerformancePeriod>('1w');
  const perfQuery = useGetPortfolioPerformanceQuery(
    { period, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const ordersQuery = useListPortfolioOrdersQuery(
    { status: 'open', portfolioId: bookId },
    { skip: !bookId && books.length !== 1, pollingInterval: visible ? 10_000 : 0, refetchOnFocus: true },
  );
  const tradesQuery = useListPortfolioTradesQuery(
    { limit: 50, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );
  const cashQuery = useListPortfolioCashMovementsQuery(
    { limit: 30, portfolioId: bookId },
    { skip: !bookId && books.length !== 1, refetchOnFocus: true },
  );

  const [createBook, createState] = useCreatePortfolioMutation();
  const [placeOrder, placeState] = usePlacePortfolioOrderMutation();
  const [cancelOrder, cancelState] = useCancelPortfolioOrderMutation();
  const [deposit, depositState] = useDepositPortfolioCashMutation();
  const [withdraw, withdrawState] = useWithdrawPortfolioCashMutation();

  const view = portfolioQuery.data;
  const positions = view?.positions ?? [];
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
    await placeOrder({
      portfolioId: bookId,
      exchange: values.exchange,
      symbol: values.symbol,
      side: values.side,
      type: 'market',
      quantity: values.quantity,
      lotMethod: values.lotMethod,
      idempotencyKey: `web-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`,
    }).unwrap();
    void message.success(
      values.side === 'buy'
        ? t('portfolio:trade.successBuy', { qty: values.quantity, symbol: values.symbol })
        : t('portfolio:trade.successSell', { qty: values.quantity, symbol: values.symbol }),
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

  return (
    <PageStack>
      <PageHeader
        eyebrow={t('portfolio:eyebrow')}
        title={t('portfolio:title')}
        subtitle={t('portfolio:subtitle')}
      />
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

      {(books.length > 0 || view) && (
        <>
          <PortfolioSummaryStrip
            view={view}
            isLoading={portfolioQuery.isLoading}
            currency={currency}
          />

          <Split>
            <PanelCard>
              <PaperTradeForm
                isSubmitting={placeState.isLoading}
                submitError={placeState.isError ? placeState.error : undefined}
                onSubmit={onTrade}
              />
            </PanelCard>
            <PanelCard>
              <PortfolioCashPanel
                isDepositing={depositState.isLoading}
                isWithdrawing={withdrawState.isLoading}
                depositError={depositState.isError ? depositState.error : undefined}
                withdrawError={withdrawState.isError ? withdrawState.error : undefined}
                onDeposit={async (amount, note) => {
                  await deposit({ amount, note, portfolioId: bookId }).unwrap();
                  void message.success(t('portfolio:cash.depositSuccess'));
                }}
                onWithdraw={async (amount, note) => {
                  await withdraw({ amount, note, portfolioId: bookId }).unwrap();
                  void message.success(t('portfolio:cash.withdrawSuccess'));
                }}
              />
            </PanelCard>
          </Split>

          <Section>
            <PortfolioEquityChart
              points={perfQuery.data?.points}
              startEquity={perfQuery.data?.startEquity}
              startAt={perfQuery.data?.startAt}
              period={period}
              onPeriodChange={setPeriod}
              isLoading={perfQuery.isLoading || perfQuery.isFetching}
              isError={perfQuery.isError}
            />
          </Section>

          <Tabs
            items={[
              {
                key: 'positions',
                label: t('portfolio:positions.title'),
                children: (
                  <PortfolioPositionsTable
                    items={positions}
                    loading={portfolioQuery.isLoading}
                    onOpen={openMarket}
                  />
                ),
              },
              {
                key: 'orders',
                label: t('portfolio:orders.title'),
                children: (
                  <PortfolioOrdersTable
                    items={ordersQuery.data?.orders ?? []}
                    loading={ordersQuery.isLoading}
                    cancelLoading={cancelState.isLoading}
                    onOpen={openMarket}
                    onCancel={(id) => {
                      void cancelOrder({ id, portfolioId: bookId })
                        .unwrap()
                        .then(() => message.success(t('portfolio:orders.cancelSuccess')))
                        .catch(() => undefined);
                    }}
                  />
                ),
              },
              {
                key: 'trades',
                label: t('portfolio:trades.title'),
                children: (
                  <PortfolioTradesTable
                    items={tradesQuery.data?.trades ?? []}
                    loading={tradesQuery.isLoading}
                    onOpen={openMarket}
                  />
                ),
              },
              {
                key: 'cash',
                label: t('portfolio:movements.title'),
                children:
                  !cashQuery.isLoading && (cashQuery.data?.movements?.length ?? 0) === 0 ? (
                    <Text variant="body" color="secondary">
                      {t('portfolio:movements.empty')}
                    </Text>
                  ) : (
                    <DataTableCard>
                      <DataTable
                        rowKey={(r) => (r as { id?: string }).id ?? Math.random().toString()}
                        size="small"
                        loading={cashQuery.isLoading}
                        pagination={false}
                        columns={cashColumns}
                        dataSource={cashQuery.data?.movements ?? []}
                      />
                    </DataTableCard>
                  ),
              },
            ]}
          />
        </>
      )}
    </PageStack>
  );
}
