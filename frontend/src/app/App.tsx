import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Providers } from './providers';
import { AppRoutes } from './routes';
import { AppJumpSearch } from './AppJumpSearch';
import { BrandMark } from '@/components/atoms/BrandMark';
import { Text } from '@/components/atoms/Text';
import { CurrencySwitcher } from '@/components/molecules/CurrencySwitcher';
import { LanguageSwitcher } from '@/components/molecules/LanguageSwitcher';
import { PageEnter } from '@/components/molecules/PageEnter';
import { WatchlistDeskTape } from './WatchlistDeskTape';
import { DeskPriceTape } from '@/components/organisms/DeskPriceTape';
import {
  AppShell,
  BrandCopy,
  BrandLink,
  BrandName,
  NavLink,
} from '@/components/templates/AppShell';
import { APP_NAME } from '@/config/constants';
import { useDeskPriceTape } from '@/libs/hooks';

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
  const tape = useDeskPriceTape();

  return (
    <AppShell
      wide
      navAriaLabel={t('nav.main')}
      tape={
        <DeskPriceTape
          source={tape.source}
          onSourceChange={tape.setSource}
          items={tape.items}
          isLoading={tape.isLoading}
          paused={tape.paused}
          sourceAriaLabel={t('tape.sourceAria', { defaultValue: 'Price tape source' })}
          tapeAriaLabel={t('tape.aria')}
          emptyLabel={t('tape.empty', { defaultValue: 'No prices yet' })}
        >
          {tape.source === 'watchlist' ? (
            <WatchlistDeskTape
              ariaLabel={t('tape.aria')}
              emptyLabel={t('tape.watchlistEmpty', { defaultValue: 'Add symbols to your watchlist' })}
              paused={tape.paused}
            />
          ) : null}
        </DeskPriceTape>
      }
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
