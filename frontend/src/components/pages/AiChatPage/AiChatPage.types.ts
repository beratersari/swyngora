export type ChatRole = 'user' | 'assistant' | 'system';

export type ChatReference = {
  title?: string;
  url: string;
  source?: 'web' | 'news' | 'x' | 'hn' | string;
  snippet?: string;
};

export type ThinkStepKind = 'status' | 'thinking' | 'tool' | 'tool_result' | 'tool_error';

export type ThinkStep = {
  id: string;
  kind: ThinkStepKind;
  text: string;
};

export type ChatMessage = {
  id: string;
  role: ChatRole;
  content: string;
  /** Tool names invoked (assistant turns). */
  tools?: string[];
  /** High-level plan lines (assistant turns). */
  thinking?: string[];
  /** Live / persisted process steps (status, thinking, tools). */
  steps?: ThinkStep[];
  /** True while the stream is still open. */
  streaming?: boolean;
  /** Public web/X/news URLs gathered by research tools. */
  references?: ChatReference[];
  /** True when this bubble is an error presentation. */
  isError?: boolean;
  createdAt: number;
};
