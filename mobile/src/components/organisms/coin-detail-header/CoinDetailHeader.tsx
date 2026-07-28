import { Pressable, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { ChevronLeft } from 'lucide-react-native';
import { Button } from '@/components/atoms/button';
import { Icon } from '@/components/atoms/icon';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { StarButton } from '@/components/molecules/star-button';
import { semanticColors } from '@/styles/tokens';
import type { CoinDetailHeaderProps } from './CoinDetailHeader.types';
import { styles } from './CoinDetailHeader.styles';

export function CoinDetailHeader({
  symbol,
  exchange,
  lastPriceLabel,
  changePercentLabel,
  changeTone,
  isLoading,
  onBack,
  watched,
  onStarPress,
  askAiLabel,
  onAskAi,
}: CoinDetailHeaderProps) {
  const { t } = useTranslation('common');
  return (
    <View style={styles.card}>
      <View style={styles.topBar}>
        <Pressable
          onPress={onBack}
          accessibilityRole="button"
          accessibilityLabel={t('actions.back')}
          style={styles.backBtn}
        >
          <Icon icon={ChevronLeft} size="md" color={semanticColors.text.secondary} />
          <Text variant="caption" color="steel">
            {t('actions.back')}
          </Text>
        </Pressable>
        {onStarPress != null ? (
          <StarButton
            watched={Boolean(watched)}
            onPress={onStarPress}
            accessibilityLabel={
              watched
                ? t('a11y.removeFavorite', { symbol })
                : t('a11y.addFavorite', { symbol })
            }
          />
        ) : null}
      </View>
      <View style={styles.top}>
        <View style={styles.left}>
          {isLoading ? (
            <Skeleton height={28} width={140} />
          ) : (
            <Text variant="h2">{symbol}</Text>
          )}
          <Text variant="caption" color="secondary" style={styles.exchange}>
            {exchange}
          </Text>
        </View>
        <View style={styles.right}>
          {isLoading ? (
            <Skeleton height={24} width={100} />
          ) : (
            <Text variant="h3">{lastPriceLabel}</Text>
          )}
          {isLoading ? (
            <Skeleton height={16} width={72} />
          ) : (
            <Text variant="label" color={changeTone}>
              {changePercentLabel}
            </Text>
          )}
        </View>
      </View>
      {onAskAi != null && askAiLabel ? (
        <View style={styles.askRow}>
          <Button label={askAiLabel} variant="secondary" onPress={onAskAi} />
        </View>
      ) : null}
    </View>
  );
}
