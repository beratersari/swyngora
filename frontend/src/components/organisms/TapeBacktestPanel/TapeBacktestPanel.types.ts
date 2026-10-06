import type { TAPE_CONDITIONS } from './constants';

export type TapeCondition = (typeof TAPE_CONDITIONS)[number];

export type TapeDraft = {
  symbol: string;
  condition: TapeCondition;
  side: string;
  exchange: string;
  days: number;
  threshold: number;
};
