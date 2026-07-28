import { View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/button';
import { MarketsFilterForm } from '@/components/organisms/markets-filter-form';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type { MarketsFilterPageProps, MarketsFilterPageViewModel } from './MarketsFilterPage.types';
import { useMarketsFilterPageViewModel } from './MarketsFilterPage.viewModel';
import { styles } from './MarketsFilterPage.styles';

function MarketsFilterPageView({ vm }: { vm: MarketsFilterPageViewModel }) {
  const { t } = useTranslation('common');
  return (
    <ScreenTemplate
      title={vm.title}
      footer={
        <View style={styles.footer}>
          <View style={styles.footerBtn}>
            <Button label={t('actions.cancel')} variant="secondary" onPress={vm.onCancel} />
          </View>
          <View style={styles.footerBtn}>
            <Button label={t('actions.apply')} onPress={vm.onApply} />
          </View>
        </View>
      }
    >
      <MarketsFilterForm
        quote={vm.quote}
        quoteOptions={vm.quoteOptions}
        onQuoteChange={vm.onQuoteChange}
        sort={vm.sort}
        order={vm.order}
        sortOptions={vm.sortOptions}
        onSortChange={vm.onSortChange}
        onOrderChange={vm.onOrderChange}
        availableTags={vm.availableTags}
        selectedTags={vm.draftTags}
        isLoadingTags={vm.isLoadingTags}
        tagsError={vm.tagsError}
        tagSearch={vm.searchTag}
        onTagSearchChange={vm.onSearchTagChange}
        onToggleTag={vm.onToggleTag}
        onClearTags={vm.onClearTags}
        onSelectAllVisible={vm.onSelectAllVisible}
        onResetAll={vm.onResetAll}
      />
    </ScreenTemplate>
  );
}

function MarketsFilterPageConnected() {
  const vm = useMarketsFilterPageViewModel();
  return <MarketsFilterPageView vm={vm} />;
}

export function MarketsFilterPage({ viewModel }: MarketsFilterPageProps = {}) {
  if (viewModel) {
    return <MarketsFilterPageView vm={viewModel} />;
  }
  return <MarketsFilterPageConnected />;
}
