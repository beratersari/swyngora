import { RefreshControl, ScrollView, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/text';
import { LanguageSwitcher } from '@/components/molecules/language-switcher';
import { QuickActionChips } from '@/components/molecules/quick-action-chips';
import { CategorySection } from '@/components/organisms/category-section';
import { DashboardSectionList } from '@/components/organisms/dashboard-section-list';
import { PumpTeaserCard } from '@/components/organisms/pump-teaser-card';
import { ScreenTemplate } from '@/components/templates/screen-template';
import { semanticColors } from '@/styles/tokens';
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
        ? t('common:status.error')
        : t('common:status.checking');

  return (
    <ScreenTemplate title={vm.title}>
      <ScrollView
        contentContainerStyle={styles.scroll}
        refreshControl={
          <RefreshControl
            refreshing={vm.isRefreshing}
            onRefresh={vm.onRefresh}
            tintColor={semanticColors.text.primary}
          />
        }
      >
        <Text variant="body" color="secondary" style={styles.intro}>
          {vm.intro}
        </Text>

        <View style={styles.quick}>
          <QuickActionChips actions={vm.quickActions} />
        </View>

        <CategorySection
          title={vm.categoriesTitle}
          actionLabel={vm.seeAllLabel}
          onAction={vm.onOpenCategories}
          tags={vm.categoryTags}
          onSelectTag={vm.onSelectCategory}
          isLoading={vm.categoriesLoading}
          errorMessage={vm.categoriesError}
          emptyMessage={vm.categoriesEmpty}
          onRetry={vm.onRetryCategories}
          retryLabel={vm.retryLabel}
          formatLabel={vm.formatCategoryLabel}
        />

        <DashboardSectionList
          title={vm.favoritesTitle}
          actionLabel={vm.favorites.length > 0 ? vm.seeAllLabel : undefined}
          onAction={vm.favorites.length > 0 ? vm.onOpenFavorites : undefined}
          rows={vm.favorites}
          isLoading={vm.favoritesLoading}
          emptyMessage={vm.favoritesEmpty}
          onPressRow={vm.onPressMarket}
          retryLabel={vm.retryLabel}
        />

        <DashboardSectionList
          title={vm.moversTitle}
          actionLabel={vm.seeAllLabel}
          onAction={vm.onOpenMoversSeeAll}
          rows={vm.movers}
          isLoading={vm.moversLoading}
          errorMessage={vm.moversError}
          emptyMessage={vm.moversEmpty}
          onPressRow={vm.onPressMarket}
          onRetry={vm.onRetryMovers}
          retryLabel={vm.retryLabel}
        />

        <DashboardSectionList
          title={vm.volumeTitle}
          actionLabel={vm.seeAllLabel}
          onAction={vm.onOpenVolumeSeeAll}
          rows={vm.volume}
          isLoading={vm.volumeLoading}
          errorMessage={vm.volumeError}
          emptyMessage={vm.volumeEmpty}
          onPressRow={vm.onPressMarket}
          onRetry={vm.onRetryVolume}
          retryLabel={vm.retryLabel}
        />

        <PumpTeaserCard
          title={vm.pumpsTitle}
          actionLabel={vm.seeAllLabel}
          onAction={vm.onOpenPumps}
          items={vm.pumps}
          isLoading={vm.pumpsLoading}
          errorMessage={vm.pumpsError}
          emptyMessage={vm.pumpsEmpty}
          disclaimer={vm.pumpsDisclaimer}
          onPressItem={vm.onPressPump}
          onRetry={vm.onRetryPumps}
          retryLabel={vm.retryLabel}
        />

        <View style={styles.footerCard}>
          <LanguageSwitcher />
          <View style={styles.row}>
            <Text variant="caption" color="secondary">
              {t('home:backendHealth')}:{' '}
              <Text
                variant="caption"
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
            </Text>
            <Text variant="caption" color="steel">
              {vm.apiBaseUrlLabel}
            </Text>
            {vm.pollingCaption ? (
              <Text variant="caption" color="steel">
                {vm.pollingCaption}
              </Text>
            ) : null}
          </View>
        </View>
      </ScrollView>
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
