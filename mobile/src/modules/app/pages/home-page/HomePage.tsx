import { View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { LanguageSwitcher } from '@/components/molecules/language-switcher';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type { HomePageProps, HomePageViewModel } from './HomePage.types';
import { useHomePageViewModel } from './HomePage.viewModel';
import { styles } from './HomePage.styles';

function HomePageView({ vm }: { vm: HomePageViewModel }) {
  const { t } = useTranslation(['home', 'common']);

  const healthLabel =
    vm.healthStatus === 'ok'
      ? vm.healthDetail
        ? t('home:healthOkDetail', { detail: vm.healthDetail })
        : t('home:healthOk')
      : vm.healthStatus === 'error'
        ? (vm.errorMessage ?? t('common:status.error'))
        : t('common:status.checking');

  return (
    <ScreenTemplate title={vm.title}>
      <Text variant="body" color="secondary">
        {t('home:intro')}
      </Text>

      <View style={styles.card}>
        <LanguageSwitcher />
      </View>

      <View style={styles.card}>
        <Text variant="label" color="secondary">
          {t('home:features')}
        </Text>
        <View style={styles.actions}>
          <Button label={t('home:openMarkets')} onPress={vm.onOpenMarkets} />
          <Button label={t('home:openPumps')} onPress={vm.onOpenPumps} variant="secondary" />
          <Button label={t('home:openAsk')} onPress={vm.onOpenAsk} variant="secondary" />
        </View>
        <Text variant="caption" color="steel">
          {t('home:pumpsHint')}
        </Text>
      </View>

      <View style={styles.card}>
        <View style={styles.row}>
          <Text variant="label" color="secondary">
            {t('home:apiBase')}
          </Text>
          <Text variant="code">{vm.apiBaseUrlLabel}</Text>
        </View>

        <View style={styles.row}>
          <Text variant="label" color="secondary">
            {t('home:backendHealth')}
          </Text>
          {vm.isLoading && !vm.healthDetail ? (
            <Skeleton height={18} width="60%" />
          ) : (
            <Text
              variant="body"
              style={
                vm.healthStatus === 'ok'
                  ? styles.badgeOk
                  : vm.healthStatus === 'error'
                    ? styles.badgeError
                    : undefined
              }
            >
              {healthLabel}
            </Text>
          )}
        </View>

        <Text variant="caption" color="secondary">
          {t('home:polling', {
            state: vm.isPollingPaused
              ? t('common:status.pollingPaused')
              : t('common:status.pollingActive'),
          })}
        </Text>

        <View style={styles.actions}>
          <Button label={t('home:retryHealth')} onPress={vm.onRetry} variant="secondary" />
        </View>
      </View>
    </ScreenTemplate>
  );
}

function HomePageConnected() {
  const vm = useHomePageViewModel();
  return <HomePageView vm={vm} />;
}

export function HomePage({ viewModel }: HomePageProps = {}) {
  if (viewModel) {
    return <HomePageView vm={viewModel} />;
  }
  return <HomePageConnected />;
}
