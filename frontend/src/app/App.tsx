import { Link } from 'react-router-dom';
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
  HeaderNav,
  HeaderSpacer,
} from './App.styles';

function AppShell() {
  const { t } = useTranslation('common');

  return (
    <AppLayout>
      <AppHeader>
        <HeaderNav>
          <Link to="/markets" style={{ textDecoration: 'none' }}>
            <Text variant="h4" color="primary" as="span">
              {t('appName', { defaultValue: APP_NAME })}
            </Text>
          </Link>
          <Text variant="label" color="secondary" as="span">
            {t('nav.markets')}
          </Text>
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
