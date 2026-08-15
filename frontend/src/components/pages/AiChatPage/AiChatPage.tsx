import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react';
import { Input } from 'antd';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/Button';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { rtkErrorMessage, streamAiChat, usePostAiChatMutation } from '@/libs/api';
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
  latestStepPreview,
  mergeThinkStep,
  nextProcessOpenMap,
  parseChatMarkdown,
  parseInlineMd,
  processPanelOpen,
  sanitizeChatReferences,
  stepsFromThinking,
  thinkStepFromEvent,
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
  ProcessIndex,
  ProcessItem,
  ProcessKind,
  ProcessList,
  ProcessPanel,
  ProcessPreview,
  ProcessText,
  ProcessTitle,
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
  const [postChat] = usePostAiChatMutation();
  const [isLoading, setIsLoading] = useState(false);
  const [processOpen, setProcessOpen] = useState<Record<string, boolean>>({});
  const threadRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const el = threadRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [messages, isLoading]);

  const mapErr = (err: unknown) =>
    rtkErrorMessage(err, {
      resource: t('ai:resource'),
      statusMessages: {
        502: t('ai:errors.unavailable'),
        503: t('ai:errors.unavailable'),
      },
    });

  const send = async (raw: string) => {
    const text = clampMessage(raw.trim(), MAX_MESSAGE_LENGTH);
    if (!canSendMessage(text, isLoading)) return;

    abortRef.current?.abort();
    const ac = new AbortController();
    abortRef.current = ac;

    setDraft('');
    const placeholder = createAssistantMessage('', { streaming: true, steps: [] });
    setMessages((prev) => [...prev, createUserMessage(text), placeholder]);
    setIsLoading(true);

    const patch = (fn: (m: ChatMessage) => ChatMessage) => {
      setMessages((prev) => prev.map((m) => (m.id === placeholder.id ? fn(m) : m)));
    };

    try {
      const finish = (res: {
        reply?: string;
        sessionId?: string;
        tools?: string[];
        thinking?: string[];
        references?: ChatMessage['references'];
      }) => {
        if (res.sessionId && res.sessionId !== sessionId) {
          persistAiSessionId(res.sessionId);
          setSessionId(res.sessionId);
        }
        patch((m) => {
          const thinking = res.thinking ?? m.thinking;
          const steps = m.steps?.length ? m.steps : stepsFromThinking(thinking);
          return {
            ...m,
            content: (res.reply ?? '').trim() || m.content || '—',
            tools: res.tools ?? m.tools,
            thinking,
            steps,
            references: sanitizeChatReferences(res.references) ?? m.references,
            streaming: false,
          };
        });
      };

      try {
        const finalEv = await streamAiChat({
          message: text,
          sessionId,
          signal: ac.signal,
          onEvent: (ev) => {
            if (ev.sessionId && ev.sessionId !== sessionId) {
              persistAiSessionId(ev.sessionId);
              setSessionId(ev.sessionId);
            }
            const step = thinkStepFromEvent(ev);
            if (step) {
              patch((m) => ({ ...m, steps: mergeThinkStep(m.steps, step), streaming: true }));
            }
            if (ev.type === 'final' && (ev.reply || ev.tools || ev.thinking || ev.references)) {
              finish({
                reply: ev.reply,
                sessionId: ev.sessionId,
                tools: ev.tools,
                thinking: ev.thinking,
                references: ev.references,
              });
            }
          },
        });
        if (finalEv.reply) {
          finish(finalEv);
        }
      } catch (streamErr) {
        if (ac.signal.aborted) return;
        const status = (streamErr as { status?: number })?.status;
        if (status === 404) {
          const res = await postChat({ message: text, sessionId }).unwrap();
          finish(res);
        } else {
          throw streamErr;
        }
      }
    } catch (err) {
      if (ac.signal.aborted) return;
      patch(() => createAssistantMessage(mapErr(err), { isError: true, id: placeholder.id }));
    } finally {
      if (abortRef.current === ac) abortRef.current = null;
      setIsLoading(false);
    }
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    void send(draft);
  };

  const onNewChat = () => {
    abortRef.current?.abort();
    setIsLoading(false);
    setMessages([]);
    setProcessOpen({});
    setDraft('');
    setSessionId(resetAiSessionId());
  };

  return (
    <PageStack>
      <PageHeader title={t('ai:title')} />

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
          const steps = m.steps ?? [];
          const tools = uniqueToolNames(m.tools);
          const processOpenNow = processPanelOpen(Boolean(m.streaming), processOpen[m.id]);
          const processPreview = processOpenNow ? '' : latestStepPreview(steps);
          return (
            <BubbleRow key={m.id} $role={m.role}>
              <Bubble $role={m.role} $error={m.isError}>
                <Text variant="caption" color="primary" as="div" style={{ opacity: 0.85 }}>
                  {m.role === 'user' ? t('ai:you') : t('ai:assistant')}
                </Text>
                {m.role === 'assistant' && !m.isError && (steps.length > 0 || m.streaming) ? (
                  <ProcessPanel
                    open={processOpenNow}
                    onToggle={(e) => {
                      const nextOpen = e.currentTarget.open;
                      setProcessOpen((prev) =>
                        nextProcessOpenMap(prev, m.id, Boolean(m.streaming), nextOpen),
                      );
                    }}
                    aria-label={t('ai:processLabel')}
                  >
                    <ProcessTitle>
                      {m.streaming
                        ? `${t('ai:processLabel')} · ${t('ai:thinkingLabel')}`
                        : t('ai:thinkingSummary', { count: steps.length })}
                      {processPreview ? (
                        <ProcessPreview title={steps[steps.length - 1]?.text}>
                          {processPreview}
                        </ProcessPreview>
                      ) : null}
                    </ProcessTitle>
                    {processOpenNow ? (
                      <ProcessList aria-live={m.streaming ? 'polite' : 'off'}>
                        {steps.length === 0 && m.streaming ? (
                          <ProcessItem $kind="status" $active>
                            <ProcessIndex>1</ProcessIndex>
                            <ProcessKind $kind="status">
                              {t('ai:stepKind.status')}
                            </ProcessKind>
                            <ProcessText>{t('ai:thinking')}</ProcessText>
                          </ProcessItem>
                        ) : (
                          steps.map((step, i) => (
                            <ProcessItem
                              key={step.id}
                              $kind={step.kind}
                              $active={Boolean(m.streaming && i === steps.length - 1)}
                            >
                              <ProcessIndex>{i + 1}</ProcessIndex>
                              <ProcessKind $kind={step.kind}>
                                {t(`ai:stepKind.${step.kind}`, { defaultValue: step.kind })}
                              </ProcessKind>
                              <ProcessText title={step.text}>{step.text}</ProcessText>
                            </ProcessItem>
                          ))
                        )}
                      </ProcessList>
                    ) : null}
                  </ProcessPanel>
                ) : null}
                {m.role === 'assistant' && !m.isError ? (
                  m.content ? (
                    <AssistantMarkdown text={m.content} />
                  ) : m.streaming ? (
                    <Text variant="body" color="secondary">
                      {t('ai:composing')}
                    </Text>
                  ) : null
                ) : (
                  <UserBody data-text-role="body">{m.content}</UserBody>
                )}
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
