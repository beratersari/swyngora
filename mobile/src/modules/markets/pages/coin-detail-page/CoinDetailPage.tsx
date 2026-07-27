import { ScrollView, View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { Text } from '@/components/atoms/text';
import { CandleChart } from '@/components/organisms/candle-chart';
import { CoinDetailHeader } from '@/components/organisms/coin-detail-header';
import { CoinDetailStats } from '@/components/organisms/coin-detail-stats';
import { IndicatorRsiPane } from '@/components/organisms/indicator-rsi-pane';
import { IntervalToolbar } from '@/components/organisms/interval-toolbar';
import { PumpEventList } from '@/components/organisms/pump-event-list';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type { CoinDetailPageProps, CoinDetailPageViewModel } from './CoinDetailPage.types';
import { useCoinDetailPageViewModel } from './CoinDetailPage.viewModel';
import { styles } from './CoinDetailPage.styles';

function CoinDetailPageView({ vm }: { vm: CoinDetailPageViewModel }) {
  return (
    <ScreenTemplate title={vm.symbol || 'Detail'}>
      <ScrollView contentContainerStyle={styles.stack}>
        <CoinDetailHeader
          symbol={vm.symbol}
          exchange={vm.exchange}
          lastPriceLabel={vm.lastPriceLabel}
          changePercentLabel={vm.changePercentLabel}
          changeTone={vm.changeTone}
          isLoading={vm.headerLoading}
          onBack={vm.onBack}
          watched={vm.watched}
          onStarPress={vm.onStarPress}
        />

        {vm.actionError ? (
          <Text variant="caption" color="error">
            {vm.actionError}
          </Text>
        ) : null}

        <CoinDetailStats
          items={vm.statsItems}
          isLoading={vm.statsLoading}
          tickerError={vm.tickerError}
          supplyError={vm.supplyError}
        />

        <IntervalToolbar
          intervals={vm.intervals}
          selected={vm.interval}
          onSelect={vm.onSelectInterval}
          isLoading={vm.intervalsLoading}
          showEma={vm.showEma}
          onToggleEma={vm.onToggleEma}
        />

        <Text variant="label" color="secondary">
          Price (OHLCV)
        </Text>
        <CandleChart
          candles={vm.candles}
          overlays={vm.candleOverlays}
          isLoading={vm.candlesLoading}
          errorMessage={vm.candlesError}
        />

        {vm.emaLatestLabels.length > 0 ? (
          <Text variant="caption" color="steel">
            {vm.emaLatestLabels.join(' · ')}
          </Text>
        ) : null}

        <IndicatorRsiPane
          data={vm.rsiPoints}
          latestRsi={vm.latestRsi}
          isLoading={vm.indicatorsLoading}
          errorMessage={vm.indicatorsError}
        />

        <PumpEventList
          rows={vm.pumpEventRows}
          isLoading={vm.pumpEventsLoading}
          errorMessage={vm.pumpEventsError}
          subtitle={vm.pumpEventsSubtitle}
          disclaimer={vm.pumpDisclaimer}
        />

        <View style={styles.retry}>
          <Button label="Retry all" variant="secondary" onPress={vm.onRetry} />
        </View>
      </ScrollView>
    </ScreenTemplate>
  );
}

function CoinDetailPageConnected() {
  const vm = useCoinDetailPageViewModel();
  return <CoinDetailPageView vm={vm} />;
}

export function CoinDetailPage({ viewModel }: CoinDetailPageProps = {}) {
  if (viewModel) {
    return <CoinDetailPageView vm={viewModel} />;
  }
  return <CoinDetailPageConnected />;
}
