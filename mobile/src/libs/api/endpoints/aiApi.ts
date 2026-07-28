import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type AiChatRequest = components['schemas']['AiChatRequest'];
export type AiChatResponse = components['schemas']['AiChatResponse'];

export type PostAiChatArg = {
  message: string;
  sessionId: string;
};

export function buildAiChatBody(arg: PostAiChatArg): AiChatRequest {
  return {
    message: arg.message.trim(),
    sessionId: arg.sessionId,
  };
}

export const aiApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    postAiChat: build.mutation<AiChatResponse, PostAiChatArg>({
      query: (arg) => ({
        url: '/api/v1/ai/chat',
        method: 'POST',
        body: buildAiChatBody(arg),
      }),
    }),
  }),
});

export const { usePostAiChatMutation } = aiApi;
