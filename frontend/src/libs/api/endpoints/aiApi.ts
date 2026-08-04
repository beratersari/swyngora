import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type AiChatRequest = components['schemas']['AiChatRequest'];
export type AiChatResponse = components['schemas']['AiChatResponse'];

export type PostAiChatArg = {
  message: string;
  sessionId?: string;
};

export const aiApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    postAiChat: build.mutation<AiChatResponse, PostAiChatArg>({
      query: (body) => ({
        url: '/api/v1/ai/chat',
        method: 'POST',
        body: {
          message: body.message,
          ...(body.sessionId ? { sessionId: body.sessionId } : {}),
        },
      }),
    }),
  }),
});

export const { usePostAiChatMutation } = aiApi;
