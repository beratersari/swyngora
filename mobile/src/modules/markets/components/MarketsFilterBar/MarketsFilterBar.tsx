import { Pressable, TextInput, View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import { semanticColors } from '@/styles/tokens';
import type { MarketsFilterBarProps } from './MarketsFilterBar.types';
import { styles } from './MarketsFilterBar.styles';

export function MarketsFilterBar({
  search,
  onSearchChange,
  quote,
  quoteOptions,
  onQuoteChange,
  availableTags,
  selectedTags,
  onToggleTag,
  onClearTags,
  sort,
  order,
  sortOptions,
  onSortChange,
  onOrderChange,
}: MarketsFilterBarProps) {
  return (
    <View style={styles.root}>
      <TextInput
        accessibilityLabel="Search markets"
        placeholder="Search pairs…"
        placeholderTextColor={semanticColors.text.disabled}
        value={search}
        onChangeText={onSearchChange}
        autoCapitalize="characters"
        autoCorrect={false}
        style={styles.search}
      />

      <View style={styles.row}>
        <Text variant="caption" color="secondary" style={styles.label}>
          Quote
        </Text>
        {quoteOptions.map((q) => {
          const active = q === quote;
          return (
            <Pressable
              key={q}
              onPress={() => onQuoteChange(q)}
              style={[styles.option, active && styles.optionActive]}
            >
              <Text variant="caption" color={active ? 'cream' : 'secondary'}>
                {q}
              </Text>
            </Pressable>
          );
        })}
      </View>

      <View style={styles.row}>
        <Text variant="caption" color="secondary" style={styles.label}>
          Sort
        </Text>
        {sortOptions.map((opt) => {
          const active = opt.value === sort;
          return (
            <Pressable
              key={opt.value}
              onPress={() => onSortChange(opt.value)}
              style={[styles.option, active && styles.optionActive]}
            >
              <Text variant="caption" color={active ? 'cream' : 'secondary'}>
                {opt.label}
              </Text>
            </Pressable>
          );
        })}
        <Pressable
          onPress={() => onOrderChange(order === 'desc' ? 'asc' : 'desc')}
          style={[styles.option, styles.optionActive]}
        >
          <Text variant="caption">{order === 'desc' ? 'Desc' : 'Asc'}</Text>
        </Pressable>
      </View>

      {availableTags.length > 0 ? (
        <View>
          <View style={styles.row}>
            <Text variant="caption" color="secondary" style={styles.label}>
              Tags
            </Text>
            {selectedTags.length > 0 ? (
              <Pressable onPress={onClearTags} style={styles.clearBtn}>
                <Text variant="caption" color="steel">
                  Clear
                </Text>
              </Pressable>
            ) : null}
          </View>
          <View style={[styles.tagsRow, styles.tagsScroll]}>
            {availableTags.slice(0, 24).map((tag) => {
              const active = selectedTags.includes(tag);
              return (
                <Pressable
                  key={tag}
                  onPress={() => onToggleTag(tag)}
                  style={[styles.option, active && styles.optionActive]}
                >
                  <Text variant="caption" color={active ? 'cream' : 'secondary'}>
                    {tag}
                  </Text>
                </Pressable>
              );
            })}
          </View>
        </View>
      ) : null}
    </View>
  );
}
