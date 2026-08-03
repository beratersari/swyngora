export type ChatBubbleRole = 'user' | 'assistant' | 'system';

export type ChatBubbleProps = {
  role: ChatBubbleRole;
  text: string;
  pending?: boolean;
  error?: string | null;
  /** Optional secondary lines under the bubble (tools). */
  metaLabels?: string[];
};
