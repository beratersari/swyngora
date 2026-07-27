import { useTranslation } from 'react-i18next';
import { Providers } from './providers';
import { AppRoutes } from './routes';
import { Text } from '@/components/atoms/Text';
import { LanguageSwitcher } from '@/components/molecules/LanguageSwitcher';
import { APP_NAME } from '@/config/constants';
import {
  AppContent,
  AppFooter,
  AppHeader,
  AppLayout,
  BrandLink,
  HeaderNav,
  HeaderSpacer,
  NavLink,
} from './App.styles';

function AppShell() {
  const { t } = useTranslation('common');

  return (
    <AppLayout>
      <AppHeader>
        <HeaderNav aria-label={t('nav.markets')}>
          <BrandLink to="/markets">
            <Text variant="h4" color="primary" as="span">
              {t('appName', { defaultValue: APP_NAME })}
            </Text>
          </BrandLink>
          <NavLink to="/markets">
            <Text variant="label" color="secondary" as="span">
              {t('nav.markets')}
            </Text>
          </NavLink>
          <NavLink to="/watchlist">
            <Text variant="label" color="secondary" as="span">
              {t('nav.watchlist')}
            </Text>
          </NavLink>
          <NavLink to="/pumps">
            <Text variant="label" color="secondary" as="span">
              {t('nav.pumps')}
            </Text>
          </NavLink>
        </HeaderNav>
        <HeaderSpacer />
        <LanguageSwitcher />
      </AppHeader>
      <AppContent>
        <AppRoutes />
      </AppContent>
      <AppFooter>
        <Text variant="caption" color="secondary">
          {t('footer.disclaimer', { appName: t('appName', { defaultValue: APP_NAME }) })}
        </Text>
      </AppFooter>
    </AppLayout>
  );
}

export function App() {
  return (
    <Providers>
      <AppShell />
    </Providers>
  );
}
