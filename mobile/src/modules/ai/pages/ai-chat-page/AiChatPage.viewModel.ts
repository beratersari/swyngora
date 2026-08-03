import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRoute } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import { useTranslation } from 'react-i18next';
import { AI_CHAT_DISCLAIMER, MAX_CHAT_MESSAGES } from '@/config/aiChatConstants';
import { rtkErrorMessage, usePostAiChatMutation } from '@/libs/api';
import {
  buildContextPrompt,
  createAssistantMessage,
  createPendingAssistantMessage,
  createUserMessage,
  getOrCreateAiSessionId,
  rotateAiSessionId,
  trimMessages,
  type ChatMessageModel,
} from '@/libs/utils';
import type { AiStackParamList } from '../../navigation';
import type { AiChatPageViewModel } from './AiChatPage.types';

function mapStatusError(
  err: unknown,
  t: (key: string, opts?: Record<string, unknown>) => string,
): string {
  if (
    err &&
    typeof err === 'object' &&
    'status' in err &&
    (err as { status?: number }).status === 503
  ) {
    return t('ai:unavailable');
  }
  if (
    err &&
    typeof err === 'object' &&
    'status' in err &&
    (err as { status?: number }).status === 502
  ) {
    return t('ai:upstreamError');
  }
  return rtkErrorMessage(err, { resource: t('ai:resourceName') });
}

export function useAiChatPageViewModel(): AiChatPageViewModel {
  const { t } = useTranslation(['ai', 'common']);
  const route = useRoute<RouteProp<AiStackParamList, 'AiChat'>>();
  const [postAiChat, postState] = usePostAiChatMutation();

  const [sessionId, setSessionId] = useState(() => getOrCreateAiSessionId());
  const [messages, setMessages] = useState<ChatMessageModel[]>([]);
  const [draft, setDraft] = useState('');
  const [bannerError, setBannerError] = useState<string | null>(null);
  const [lastFailedText, setLastFailedText] = useState<string | null>(null);
  const [disclaimer, setDisclaimer] = useState(AI_CHAT_DISCLAIMER);

  const appliedContextKey = useRef<string | null>(null);

  // Apply navigation context once per param snapshot (user can edit draft).
  useEffect(() => {
    const params = route.params;
    if (!params) return;
    const key = JSON.stringify({
      d: params.draft ?? '',
      e: params.exchange ?? '',
      s: params.symbol ?? '',
      i: params.interval ?? '',
    });
    if (appliedContextKey.current === key) return;
    appliedContextKey.current = key;
    const built = buildContextPrompt({
      draft: params.draft,
      exchange: params.exchange,
      symbol: params.symbol,
      interval: params.interval,
    });
    if (built) {
      setDraft(built);
    }
  }, [route.params]);

  const sendText = useCallback(
    async (text: string) => {
      const trimmed = text.trim();
      if (!trimmed || postState.isLoading) return;

      setBannerError(null);
      setLastFailedText(null);
      setDraft('');

      const userMsg = createUserMessage(trimmed);
      const pending = createPendingAssistantMessage();
      setMessages((prev) =>
        trimMessages([...prev, userMsg, pending], MAX_CHAT_MESSAGES),
      );

      try {
        const res = await postAiChat({
          message: trimmed,
          sessionId,
        }).unwrap();

        const assistant = createAssistantMessage(res.reply, {
          tools: res.tools,
          thinking: res.thinking,
        });
        if (res.note?.trim()) {
          setDisclaimer(res.note.trim());
        }
        if (res.sessionId?.trim()) {
          setSessionId(res.sessionId.trim());
        }
        setMessages((prev) => {
          const withoutPending = prev.filter((m) => m.id !== pending.id);
          return trimMessages([...withoutPending, assistant], MAX_CHAT_MESSAGES);
        });
      } catch (err) {
        const msg = mapStatusError(err, t as (k: string, o?: Record<string, unknown>) => string);
        setLastFailedText(trimmed);
        setBannerError(msg);
        setMessages((prev) => {
          const withoutPending = prev.filter((m) => m.id !== pending.id);
          // Attach error to last user message if present
          const next = withoutPending.map((m, idx) =>
            idx === withoutPending.length - 1 && m.role === 'user'
              ? { ...m, error: msg }
              : m,
          );
          return next;
        });
      }
    },
    [postAiChat, postState.isLoading, sessionId, t],
  );

  const onSend = useCallback(() => {
    void sendText(draft);
  }, [draft, sendText]);

  const onRetryLast = useCallback(() => {
    if (lastFailedText) {
      void sendText(lastFailedText);
    }
  }, [lastFailedText, sendText]);

  const onNewChat = useCallback(() => {
    const next = rotateAiSessionId();
    setSessionId(next);
    setMessages([]);
    setDraft('');
    setBannerError(null);
    setLastFailedText(null);
    setDisclaimer(AI_CHAT_DISCLAIMER);
    appliedContextKey.current = null;
  }, []);

  const listMessages = useMemo(
    () =>
      messages.map((m) => ({
        id: m.id,
        role: m.role,
        text: m.pending
          ? t('ai:thinking')
          : m.thinking && m.thinking.length > 0 && m.role === 'assistant'
            ? m.text
            : m.text,
        pending: m.pending,
        error: m.error,
        tools: m.tools,
      })),
    [messages, t],
  );

  return {
    title: t('ai:title'),
    draft,
    onChangeDraft: setDraft,
    onSend,
    onNewChat,
    onRetryLast,
    messages: listMessages,
    isSending: postState.isLoading,
    sendDisabled: postState.isLoading || draft.trim().length === 0,
    bannerError,
    emptyTitle: t('ai:emptyTitle'),
    emptyMessage: t('ai:emptyMessage'),
    disclaimer: t('ai:disclaimer', { defaultValue: disclaimer }) || disclaimer,
    thinkingLabel: t('ai:thinking'),
    placeholder: t('ai:placeholder'),
    sendLabel: t('ai:send'),
    newChatLabel: t('ai:newChat'),
    toolsLabel: t('ai:toolsUsed'),
    retryLabel: t('common:actions.retry'),
    showRetry: Boolean(lastFailedText) && !postState.isLoading,
  };
}
