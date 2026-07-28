import type { ChatMessageListItem } from '@/components/organisms/chat-message-list';

export type AiChatPageViewModel = {
  title: string;
  draft: string;
  onChangeDraft: (text: string) => void;
  onSend: () => void;
  onNewChat: () => void;
  onRetryLast: () => void;
  messages: ChatMessageListItem[];
  isSending: boolean;
  sendDisabled: boolean;
  bannerError: string | null;
  emptyTitle: string;
  emptyMessage: string;
  disclaimer: string;
  /** Shown while a request is in flight (slow local Ollama). */
  thinkingLabel?: string;
  placeholder: string;
  sendLabel: string;
  newChatLabel: string;
  toolsLabel: string;
  retryLabel: string;
  showRetry: boolean;
};

export type AiChatPageProps = {
  viewModel?: AiChatPageViewModel;
};
