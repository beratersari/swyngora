import { StarFilled, StarOutlined } from '@ant-design/icons';
import { StarButton } from './WatchStar.styles';
import type { WatchStarProps } from './WatchStar.types';

/** Watchlist star with a short pop when added. */
export function WatchStar({
  watched,
  loading = false,
  disabled = false,
  addLabel,
  removeLabel,
  onClick,
}: WatchStarProps) {
  return (
    <StarButton
      type="text"
      size="small"
      $watched={watched}
      loading={loading}
      disabled={disabled}
      aria-label={watched ? removeLabel : addLabel}
      aria-pressed={watched}
      icon={watched ? <StarFilled /> : <StarOutlined />}
      onClick={onClick}
    />
  );
}
