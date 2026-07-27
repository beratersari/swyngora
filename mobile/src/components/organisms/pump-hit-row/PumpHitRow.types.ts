export type PumpHitRowViewModel = {
  id: string;
  symbol: string;
  exchange: string;
  bestReturnLabel: string;
  bestReturnTone: 'success' | 'error' | 'secondary';
  eventsLabel: string;
  metaLabel: string;
};

export type PumpHitRowProps = {
  row: PumpHitRowViewModel;
  onPress?: (exchange: string, symbol: string) => void;
};
