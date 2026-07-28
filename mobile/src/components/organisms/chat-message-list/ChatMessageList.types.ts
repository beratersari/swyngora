export type ChatMessageListItem = {
  id: string;
  role: 'user' | 'assistant' | 'system';
  text: string;
  pending?: boolean;
  error?: string | null;
  tools?: string[];
};

export type ChatMessageListProps = {
  messages: ChatMessageListItem[];
  emptyTitle?: string;
  emptyMessage?: string | null;
  bannerError?: string | null;
  toolsLabel?: string;
};
