import { Providers } from './providers';
import { AppRoutes } from './routes';
import { Text } from '@/components/atoms/Text';
import { APP_NAME } from '@/config/constants';
import { AppContent, AppFooter, AppHeader, AppLayout } from './App.styles';

export function App() {
  return (
    <Providers>
      <AppLayout>
        <AppHeader>
          <Text variant="h4" color="cream" as="span">
            {APP_NAME}
          </Text>
          <Text variant="label" color="steel" as="span">
            Markets
          </Text>
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
    </Providers>
  );
}
