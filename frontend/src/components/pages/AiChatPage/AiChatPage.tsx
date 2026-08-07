import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react';
import { Input } from 'antd';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/Button';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { rtkErrorMessage, usePostAiChatMutation } from '@/libs/api';
import {
  getOrCreateAiSessionId,
  persistAiSessionId,
  resetAiSessionId,
} from '@/libs/utils';
import { MAX_MESSAGE_LENGTH, SUGGESTION_KEYS } from './AiChatPage.constants';
import {
  canSendMessage,
  clampMessage,
  createAssistantMessage,
  createUserMessage,
  parseChatMarkdown,
  parseInlineMd,
  sanitizeThinkingLines,
  uniqueToolNames,
} from './AiChatPage.helpers';
import type { ChatMessage } from './AiChatPage.types';
import {
  Bubble,
  BubbleRow,
  Composer,
  ComposerRow,
  DisclaimerBanner,
  DisclaimerBody,
  DisclaimerIcon,
  EmptyState,
  MdCode,
  MdList,
  MdOl,
  MdP,
  MdPre,
  MdStack,
  MdStrong,
  MetaChip,
  MetaLabel,
  MetaRow,
  PageStack,
  Suggestions,
  Thread,
  ToolbarRow,
  TraceDetails,
  TraceList,
  UserBody,
  RefItem,
  RefLink,
  RefList,
  RefUrl,
} from './AiChatPage.styles';

function renderInline(text: string): ReactNode[] {
  return parseInlineMd(text).map((tok, i) => {
    if (tok.t === 'strong') return <MdStrong key={i}>{tok.v}</MdStrong>;
    if (tok.t === 'code') return <MdCode key={i}>{tok.v}</MdCode>;
    return <span key={i}>{tok.v}</span>;
  });
}

function AssistantMarkdown({ text }: { text: string }) {
  const blocks = parseChatMarkdown(text);
  if (blocks.length === 0) return <UserBody data-text-role="body">{text}</UserBody>;
  return (
    <MdStack data-text-role="body">
      {blocks.map((b, i) => {
        if (b.type === 'ul') {
          return (
            <MdList key={i}>
              {b.items.map((item) => (
                <li key={item}>{renderInline(item)}</li>
              ))}
            </MdList>
          );
        }
        if (b.type === 'ol') {
          return (
            <MdOl key={i}>
              {b.items.map((item) => (
                <li key={item}>{renderInline(item)}</li>
              ))}
            </MdOl>
          );
        }
        if (b.type === 'pre') {
          return <MdPre key={i}>{b.text}</MdPre>;
        }
        return <MdP key={i}>{renderInline(b.text)}</MdP>;
      })}
    </MdStack>
  );
}

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
      if (res.sessionId && res.sessionId !== sessionId) {
        persistAiSessionId(res.sessionId);
        setSessionId(res.sessionId);
      }
      setMessages((prev) => [
        ...prev,
        createAssistantMessage(res.reply ?? '', {
          tools: res.tools ?? undefined,
          thinking: res.thinking ?? undefined,
          references: res.references ?? undefined,
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
      <PageHeader eyebrow={t('ai:eyebrow')} title={t('ai:title')} subtitle={t('ai:subtitle')} />

      <DisclaimerBanner role="note">
        <DisclaimerIcon aria-hidden />
        <DisclaimerBody>{t('ai:disclaimer')}</DisclaimerBody>
      </DisclaimerBanner>

      <ToolbarRow>
        <Text variant="caption" color="primary">
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

        {messages.map((m) => {
          const think = sanitizeThinkingLines(m.thinking, m.content);
          const tools = uniqueToolNames(m.tools);
          return (
            <BubbleRow key={m.id} $role={m.role}>
              <Bubble $role={m.role} $error={m.isError}>
                <Text variant="caption" color="primary" as="div" style={{ opacity: 0.85 }}>
                  {m.role === 'user' ? t('ai:you') : t('ai:assistant')}
                </Text>
                {m.role === 'assistant' && !m.isError ? (
                  <AssistantMarkdown text={m.content} />
                ) : (
                  <UserBody data-text-role="body">{m.content}</UserBody>
                )}
                {think.length > 0 ? (
                  <TraceDetails>
                    <summary>
                      {t('ai:thinkingSummary', { count: think.length, defaultValue: `Thinking · ${think.length}` })}
                    </summary>
                    <TraceList>
                      {think.map((line) => (
                        <li key={line}>{line}</li>
                      ))}
                    </TraceList>
                  </TraceDetails>
                ) : null}
                {tools.length > 0 ? (
                  <MetaRow>
                    <MetaLabel>{t('ai:toolsLabel', { defaultValue: 'Tools' })}</MetaLabel>
                    {tools.map((tool) => (
                      <MetaChip key={tool} $kind="tool" title={tool}>
                        {tool}
                      </MetaChip>
                    ))}
                  </MetaRow>
                ) : null}
                {m.references && m.references.length > 0 ? (
                  <MetaRow>
                    <MetaLabel>{t('ai:sourcesLabel', { defaultValue: 'Sources' })}</MetaLabel>
                    <RefList>
                      {m.references.map((ref, i) => (
                        <RefItem key={ref.url}>
                          <RefLink href={ref.url} target="_blank" rel="noopener noreferrer">
                            {i + 1}. {ref.title || ref.url}
                          </RefLink>
                          <RefUrl>{ref.url}</RefUrl>
                        </RefItem>
                      ))}
                    </RefList>
                  </MetaRow>
                ) : null}
              </Bubble>
            </BubbleRow>
          );
        })}

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
