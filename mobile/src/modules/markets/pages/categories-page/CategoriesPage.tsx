import { ScrollView, View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { Text } from '@/components/atoms/text';
import { SearchField } from '@/components/molecules/search-field';
import { CategoryChipGrid } from '@/components/organisms/category-chip-grid';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type { CategoriesPageProps, CategoriesPageViewModel } from './CategoriesPage.types';
import { useCategoriesPageViewModel } from './CategoriesPage.viewModel';
import { styles } from './CategoriesPage.styles';

function CategoriesPageView({ vm }: { vm: CategoriesPageViewModel }) {
  return (
    <ScreenTemplate title={vm.title}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.headerActions}>
          <Button label={vm.backLabel} variant="secondary" onPress={vm.onBack} />
        </View>

        <View style={styles.searchBlock}>
          <SearchField
            value={vm.search}
            onChangeText={vm.onSearchChange}
            placeholder={vm.searchPlaceholder}
            accessibilityLabel={vm.searchPlaceholder}
            autoCapitalize="none"
          />
          {vm.isSearchDebouncing ? (
            <Text variant="caption" color="steel">
              …
            </Text>
          ) : null}
        </View>

        {vm.errorMessage ? (
          <View style={styles.errorBlock}>
            <Text variant="body" color="error">
              {vm.errorMessage}
            </Text>
            <Button label={vm.retryLabel} variant="secondary" onPress={vm.onRetry} />
          </View>
        ) : (
          <CategoryChipGrid
            featuredTitle={vm.featuredTags.length > 0 ? vm.featuredTitle : undefined}
            featuredTags={vm.featuredTags}
            allTitle={vm.allTitle}
            tags={vm.tags}
            selectedTag={vm.selectedTag}
            onSelectTag={vm.onSelectTag}
            isLoading={vm.isLoading}
            emptyMessage={vm.emptyMessage}
            formatLabel={vm.formatLabel}
          />
        )}
      </ScrollView>
    </ScreenTemplate>
  );
}

function CategoriesPageConnected() {
  const vm = useCategoriesPageViewModel();
  return <CategoriesPageView vm={vm} />;
}

export function CategoriesPage({ viewModel }: CategoriesPageProps = {}) {
  if (viewModel) {
    return <CategoriesPageView vm={viewModel} />;
  }
  return <CategoriesPageConnected />;
}
