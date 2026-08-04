export type ChatRole = 'user' | 'assistant' | 'system';

export type ChatMessage = {
  id: string;
  role: ChatRole;
  content: string;
  /** Tool names invoked (assistant turns). */
  tools?: string[];
  /** High-level plan lines (assistant turns). */
  thinking?: string[];
  /** True when this bubble is an error presentation. */
  isError?: boolean;
  createdAt: number;
};
