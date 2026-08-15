import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Providers } from './providers';
import { AppRoutes } from './routes';
import { AppJumpSearch } from './AppJumpSearch';
import { BrandMark } from '@/components/atoms/BrandMark';
import { Text } from '@/components/atoms/Text';
import { CurrencySwitcher } from '@/components/molecules/CurrencySwitcher';
import { GlobalStatsBar } from '@/components/molecules/GlobalStatsBar';
import { LanguageSwitcher } from '@/components/molecules/LanguageSwitcher';
import { PageEnter } from '@/components/molecules/PageEnter';
import {
  AppShell,
  BrandCopy,
  BrandLink,
  BrandName,
  NavLink,
} from '@/components/templates/AppShell';
import { APP_NAME, DEFAULT_SPOT_POLL_MS } from '@/config/constants';
import { useListSpotMarketsQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { formatChangePercent, formatCompactUsd, formatPrice } from '@/libs/utils';

const NAV_ITEMS = [
  { to: '/markets', key: 'nav.markets' as const },
  { to: '/heatmap', key: 'nav.heatmap' as const },
  { to: '/watchlist', key: 'nav.watchlist' as const },
  { to: '/portfolio', key: 'nav.portfolio' as const },
  { to: '/signals', key: 'nav.signals' as const },
  { to: '/pumps', key: 'nav.pumps' as const },
  { to: '/alerts', key: 'nav.alerts' as const },
  { to: '/compare', key: 'nav.compare' as const },
  { to: '/ai', key: 'nav.ai' as const },
];

function DeskShell() {
  const { t } = useTranslation('common');
  const location = useLocation();
  const visible = useDocumentVisible();
  const tapeQuery = useListSpotMarketsQuery(
    {
      exchange: 'binance',
      quote: 'USDT',
      sort: 'quoteVolume',
      order: 'desc',
      limit: 50,
      offset: 0,
      status: 'TRADING',
    },
    { pollingInterval: visible ? DEFAULT_SPOT_POLL_MS : 0, refetchOnFocus: true },
  );

  const stats = useMemo(() => {
    const rows = tapeQuery.data?.items ?? [];
    const total = tapeQuery.data?.total ?? rows.length;
    let vol = 0;
    for (const row of rows) {
      const n = Number(row.quoteVolume);
      if (Number.isFinite(n)) vol += n;
    }
    const btc = rows.find((row) => (row.symbol ?? '').toUpperCase() === 'BTCUSDT');
    const chg = btc ? Number(btc.priceChangePercent) : NaN;
    return {
      coinCount: total,
      volumeLabel: `$${formatCompactUsd(vol)}`,
      btcPrice: btc ? `$${formatPrice(btc.lastPrice)}` : undefined,
      btcChange: Number.isFinite(chg) ? formatChangePercent(chg) : undefined,
      btcUp: Number.isFinite(chg) ? chg >= 0 : undefined,
    };
  }, [tapeQuery.data?.items, tapeQuery.data?.total]);

  return (
    <AppShell
      wide
      navAriaLabel={t('nav.main')}
      brand={
        <BrandLink to="/markets">
          <BrandMark size={28} />
          <BrandCopy>
            <BrandName>{t('appName', { defaultValue: APP_NAME })}</BrandName>
          </BrandCopy>
        </BrandLink>
      }
      nav={
        <>
          {NAV_ITEMS.map((item) => (
            <NavLink key={item.to} to={item.to}>
              {t(item.key)}
            </NavLink>
          ))}
        </>
      }
      tools={
        <>
          <AppJumpSearch />
          <CurrencySwitcher className="desk-fx" />
          <LanguageSwitcher className="desk-lang" />
        </>
      }
      banner={<GlobalStatsBar {...stats} />}
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
