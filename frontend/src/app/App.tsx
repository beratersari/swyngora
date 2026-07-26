import { Link } from 'react-router-dom';
import { Providers } from './providers';
import { AppRoutes } from './routes';
import { Text } from '@/components/atoms/Text';
import { APP_NAME } from '@/config/constants';
import { AppContent, AppFooter, AppHeader, AppLayout, HeaderNav } from './App.styles';

function AppShell() {
  return (
    <AppLayout>
      <AppHeader>
        <HeaderNav>
          <Link to="/markets" style={{ textDecoration: 'none' }}>
            <Text variant="h4" color="cream" as="span">
              {APP_NAME}
            </Text>
          </Link>
          <Text variant="label" color="steel" as="span">
            Markets
          </Text>
        </HeaderNav>
      </AppHeader>
      <AppContent>
        <AppRoutes />
      </AppContent>
      <AppFooter>
        <Text variant="caption" color="steel">
          {APP_NAME} · informational market data · not financial advice
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
