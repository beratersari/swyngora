export type ChatRole = 'user' | 'assistant' | 'system';

export type ChatReference = {
  title?: string;
  url: string;
  source?: 'web' | 'news' | 'x' | 'hn' | string;
  snippet?: string;
};

export type ChatMessage = {
  id: string;
  role: ChatRole;
  content: string;
  /** Tool names invoked (assistant turns). */
  tools?: string[];
  /** High-level plan lines (assistant turns). */
  thinking?: string[];
  /** Public web/X/news URLs gathered by research tools. */
  references?: ChatReference[];
  /** True when this bubble is an error presentation. */
  isError?: boolean;
  createdAt: number;
};
