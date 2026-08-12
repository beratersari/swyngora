export type ConnectionStatusKind = 'live' | 'degraded' | 'offline' | 'paused' | 'loading';

export type ConnectionStatusProps = {
  status: ConnectionStatusKind;
  label: string;
};
