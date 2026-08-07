export type ConnectionStatusKind = 'live' | 'offline' | 'paused' | 'loading';

export type ConnectionStatusProps = {
  status: ConnectionStatusKind;
  label: string;
};
