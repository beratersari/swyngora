import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Alert, Input, Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/Button';
import { Text } from '@/components/atoms/Text';
import { rtkErrorMessage, usePostAiChatMutation } from '@/libs/api';
import { getOrCreateAiSessionId, resetAiSessionId } from '@/libs/utils';
import { MAX_MESSAGE_LENGTH, SUGGESTION_KEYS } from './AiChatPage.constants';
import {
  canSendMessage,
  clampMessage,
  createAssistantMessage,
  createUserMessage,
} from './AiChatPage.helpers';
import type { ChatMessage } from './AiChatPage.types';
import {
  Bubble,
  BubbleRow,
  Composer,
  ComposerRow,
  EmptyState,
  MetaRow,
  PageIntro,
  PageStack,
  Suggestions,
  Thread,
  ToolbarRow,
} from './AiChatPage.styles';

export function AiChatPage() {
  const { t } = useTranslation(['ai', 'common']);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [draft, setDraft] = useState('');
  const [sessionId, setSessionId] = useState(() => getOrCreateAiSessionId());
  const [postChat, { isLoading }] = usePostAiChatMutation();
  const threadRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = threadRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [messages, isLoading]);

  const send = async (raw: string) => {
    const text = clampMessage(raw.trim(), MAX_MESSAGE_LENGTH);
    if (!canSendMessage(text, isLoading)) return;

    setDraft('');
    setMessages((prev) => [...prev, createUserMessage(text)]);

    try {
      const res = await postChat({ message: text, sessionId }).unwrap();
      setMessages((prev) => [
        ...prev,
        createAssistantMessage(res.reply ?? '', {
          tools: res.tools ?? undefined,
          thinking: res.thinking ?? undefined,
        }),
      ]);
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        createAssistantMessage(
          rtkErrorMessage(err, {
            resource: t('ai:resource'),
            statusMessages: {
              502: t('ai:errors.unavailable'),
              503: t('ai:errors.unavailable'),
            },
          }),
          { isError: true },
        ),
      ]);
    }
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    void send(draft);
  };

  const onNewChat = () => {
    setMessages([]);
    setDraft('');
    setSessionId(resetAiSessionId());
  };

  return (
    <PageStack>
      <PageIntro>
        <Text variant="h2" color="primary">
          {t('ai:title')}
        </Text>
        <Text variant="body" color="secondary">
          {t('ai:subtitle')}
        </Text>
      </PageIntro>

      <Alert type="info" showIcon message={t('ai:disclaimer')} />

      <ToolbarRow>
        <Text variant="caption" color="secondary">
          {t('ai:sessionLabel')}: {sessionId}
        </Text>
        <Button type="default" size="small" onClick={onNewChat} disabled={isLoading}>
          {t('ai:newChat')}
        </Button>
      </ToolbarRow>

      <Thread ref={threadRef} role="log" aria-live="polite" aria-relevant="additions">
        {messages.length === 0 && !isLoading ? (
          <EmptyState>
            <Text variant="body" color="secondary">
              {t('ai:emptyHint')}
            </Text>
            <Suggestions>
              {SUGGESTION_KEYS.map((key) => (
                <Button
                  key={key}
                  type="default"
                  size="small"
                  onClick={() => void send(t(`ai:${key}`))}
                >
                  {t(`ai:${key}`)}
                </Button>
              ))}
            </Suggestions>
          </EmptyState>
        ) : null}

        {messages.map((m) => (
          <BubbleRow key={m.id} $role={m.role}>
            <Bubble $role={m.role} $error={m.isError}>
              <Text variant="caption" color="secondary" as="div">
                {m.role === 'user' ? t('ai:you') : t('ai:assistant')}
              </Text>
              <Text variant="body" color="primary" as="div">
                {m.content}
              </Text>
              {m.thinking && m.thinking.length > 0 ? (
                <MetaRow>
                  {m.thinking.map((line) => (
                    <Tag key={line} color="blue">
                      {line}
                    </Tag>
                  ))}
                </MetaRow>
              ) : null}
              {m.tools && m.tools.length > 0 ? (
                <MetaRow>
                  {m.tools.map((tool) => (
                    <Tag key={tool} color="green">
                      {tool}
                    </Tag>
                  ))}
                </MetaRow>
              ) : null}
            </Bubble>
          </BubbleRow>
        ))}

        {isLoading ? (
          <BubbleRow $role="assistant">
            <Bubble $role="assistant">
              <Text variant="body" color="secondary">
                {t('ai:thinking')}
              </Text>
            </Bubble>
          </BubbleRow>
        ) : null}
      </Thread>

      <Composer onSubmit={onSubmit}>
        <Input.TextArea
          value={draft}
          onChange={(e) => setDraft(clampMessage(e.target.value, MAX_MESSAGE_LENGTH))}
          placeholder={t('ai:placeholder')}
          autoSize={{ minRows: 2, maxRows: 6 }}
          maxLength={MAX_MESSAGE_LENGTH}
          disabled={isLoading}
          onPressEnter={(e) => {
            if (!e.shiftKey) {
              e.preventDefault();
              void send(draft);
            }
          }}
          aria-label={t('ai:placeholder')}
        />
        <ComposerRow>
          <Button
            type="primary"
            htmlType="submit"
            pending={isLoading}
            disabled={!canSendMessage(draft, isLoading)}
          >
            {t('ai:send')}
          </Button>
          <Text variant="caption" color="secondary">
            {t('ai:enterHint')}
          </Text>
        </ComposerRow>
      </Composer>
    </PageStack>
  );
}
