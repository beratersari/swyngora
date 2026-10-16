import type { AiChatResponse } from '@/libs/api';

export type AiSituationRead = NonNullable<AiChatResponse['reads']>[number];

export type AiSituationEntry = NonNullable<AiSituationRead['entries']>[number];

export type ChatSituationListProps = {
  reads: AiSituationRead[];
};
