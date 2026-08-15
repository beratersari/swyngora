import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Providers } from './providers';
import { AppRoutes } from './routes';
import { AppJumpSearch } from './AppJumpSearch';
import {
  AppstoreOutlined,
  BellOutlined,
  FireOutlined,
  LineChartOutlined,
  RobotOutlined,
  StarOutlined,
  SwapOutlined,
  ThunderboltOutlined,
  WalletOutlined,
} from '@ant-design/icons';
import { BrandMark } from '@/components/atoms/BrandMark';
import { Text } from '@/components/atoms/Text';
import { ConnectionStatus, type ConnectionStatusKind } from '@/components/molecules/ConnectionStatus';
import { CurrencySwitcher } from '@/components/molecules/CurrencySwitcher';
import { LanguageSwitcher } from '@/components/molecules/LanguageSwitcher';
import { PageEnter } from '@/components/molecules/PageEnter';
import { TickerTape, toTickerTapeItem } from '@/components/molecules/TickerTape';
import {
  AppShell,
  BrandCopy,
  BrandLink,
  BrandName,
  BrandTagline,
  NavLink,
} from '@/components/templates/AppShell';
import { APP_NAME, DEFAULT_SPOT_POLL_MS } from '@/config/constants';
import { useGetHealthQuery, useListSpotMarketsQuery } from '@/libs/api';
import { useDisplayCurrency, useDocumentVisible } from '@/libs/hooks';
import { useRealtimeConnected } from '@/libs/realtime';

const NAV_ITEMS = [
  { to: '/markets', key: 'nav.markets' as const, icon: <LineChartOutlined /> },
  { to: '/heatmap', key: 'nav.heatmap' as const, icon: <AppstoreOutlined /> },
  { to: '/watchlist', key: 'nav.watchlist' as const, icon: <StarOutlined /> },
  { to: '/portfolio', key: 'nav.portfolio' as const, icon: <WalletOutlined /> },
  { to: '/signals', key: 'nav.signals' as const, icon: <ThunderboltOutlined /> },
  { to: '/pumps', key: 'nav.pumps' as const, icon: <FireOutlined /> },
  { to: '/alerts', key: 'nav.alerts' as const, icon: <BellOutlined /> },
  { to: '/compare', key: 'nav.compare' as const, icon: <SwapOutlined /> },
  { to: '/ai', key: 'nav.ai' as const, icon: <RobotOutlined /> },
];

function DeskShell() {
  const { t } = useTranslation('common');
  const location = useLocation();
  const visible = useDocumentVisible();
  const display = useDisplayCurrency();
  const health = useGetHealthQuery(undefined, {
    pollingInterval: visible ? 15_000 : 0,
    refetchOnFocus: true,
  });
  const tapeQuery = useListSpotMarketsQuery(
    {
      exchange: 'binance',
      quote: 'USDT',
      sort: 'quoteVolume',
      order: 'desc',
      limit: 12,
      offset: 0,
      status: 'TRADING',
    },
    { pollingInterval: visible ? DEFAULT_SPOT_POLL_MS : 0, refetchOnFocus: true },
  );

  const tapeItems = useMemo(() => {
    const rows = tapeQuery.data?.items ?? [];
    return rows
      .map((row) =>
        toTickerTapeItem(
          { ...row, exchange: 'binance' },
          { currency: display.currency, rates: display.rates },
        ),
      )
      .filter((item): item is NonNullable<typeof item> => item != null);
  }, [tapeQuery.data?.items, display.currency, display.rates]);

  const wsConnected = useRealtimeConnected();
  let connection: ConnectionStatusKind = 'loading';
  if (!visible) connection = 'paused';
  else if (health.isError) connection = 'offline';
  else if (health.isSuccess || health.data) {
    // "Live" only when the market stream is up; API-only is "Delayed".
    connection = wsConnected ? 'live' : 'degraded';
  }

  return (
    <AppShell
      wide
      flush={location.pathname.startsWith('/heatmap')}
      navAriaLabel={t('nav.main')}
      brand={
        <BrandLink to="/markets">
          <BrandMark size={22} />
          <BrandCopy>
            <BrandName>{t('appName', { defaultValue: APP_NAME })}</BrandName>
            <BrandTagline>{t('brand.tagline', { defaultValue: 'Terminal' })}</BrandTagline>
          </BrandCopy>
        </BrandLink>
      }
      nav={
        <>
          {NAV_ITEMS.map((item) => (
            <NavLink key={item.to} to={item.to}>
              {item.icon}
              {t(item.key)}
            </NavLink>
          ))}
        </>
      }
      tools={
        <>
          <AppJumpSearch />
          <ConnectionStatus status={connection} label={t(`status.connection.${connection}`)} />
          <CurrencySwitcher className="desk-fx" />
          <LanguageSwitcher className="desk-lang" />
        </>
      }
      banner={
        tapeItems.length > 0 ? (
          <TickerTape items={tapeItems} ariaLabel={t('tape.aria')} paused={!visible} />
        ) : null
      }
      footer={
        <>
          <Text variant="caption" color="secondary">
            {t('footer.venues')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('footer.disclaimer', { appName: t('appName', { defaultValue: APP_NAME }) })}
          </Text>
        </>
      }
    >
      <PageEnter motionKey={location.pathname}>
        <AppRoutes />
      </PageEnter>
    </AppShell>
  );
}

export function App() {
  return (
    <Providers>
      <DeskShell />
    </Providers>
  );
}
