import type { MouseEvent } from 'react';

export type WatchStarProps = {
  watched: boolean;
  loading?: boolean;
  disabled?: boolean;
  addLabel: string;
  removeLabel: string;
  onClick: (event: MouseEvent<HTMLButtonElement>) => void;
};
