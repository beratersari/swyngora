import { View } from 'react-native';
import { Languages } from 'lucide-react-native';
import { Icon } from '@/components/atoms/icon';
import { Text } from '@/components/atoms/text';
import { ChipGroup } from '@/components/molecules/chip-group';
import { useLocale } from '@/libs/i18n';
import { semanticColors } from '@/styles/tokens';
import type { LanguageSwitcherProps } from './LanguageSwitcher.types';
import { styles } from './LanguageSwitcher.styles';

/**
 * Locale switcher — persists via i18next detector (localStorage on web).
 * Extend SUPPORTED_LOCALES + locale JSON to add languages.
 */
export function LanguageSwitcher({ label }: LanguageSwitcherProps = {}) {
  const { locale, options, setLocale, t } = useLocale();

  return (
    <View style={styles.root} accessibilityLabel={label ?? t('language.label')}>
      <View style={styles.labelRow}>
        <Icon icon={Languages} size="sm" color={semanticColors.text.secondary} />
        <Text variant="caption" color="secondary">
          {label ?? t('language.label')}
        </Text>
      </View>
      <ChipGroup
        options={options.map((o) => ({ value: o.value, label: o.label }))}
        selected={locale}
        onSelect={(code) => {
          void setLocale(code);
        }}
        mode="single"
        shape="pill"
        horizontalScroll
      />
    </View>
  );
}
