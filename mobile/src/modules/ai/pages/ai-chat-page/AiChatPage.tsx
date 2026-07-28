import { ScrollView, View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { ChatComposer } from '@/components/molecules/chat-composer';
import { ChatDisclaimer } from '@/components/molecules/chat-disclaimer';
import { ChatMessageList } from '@/components/organisms/chat-message-list';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type { AiChatPageProps, AiChatPageViewModel } from './AiChatPage.types';
import { useAiChatPageViewModel } from './AiChatPage.viewModel';
import { styles } from './AiChatPage.styles';

function AiChatPageView({ vm }: { vm: AiChatPageViewModel }) {
  return (
    <ScreenTemplate title={vm.title}>
      <View style={styles.root}>
        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
        >
          <ChatMessageList
            messages={vm.messages}
            emptyTitle={vm.emptyTitle}
            emptyMessage={vm.emptyMessage}
            bannerError={vm.bannerError}
            toolsLabel={vm.toolsLabel}
          />
        </ScrollView>

        <View style={styles.footer}>
          {vm.isSending ? (
            <ChatDisclaimer text={vm.thinkingLabel ?? 'Thinking… local AI may take 1–3 minutes'} />
          ) : (
            <ChatDisclaimer text={vm.disclaimer} />
          )}
          <ChatComposer
            value={vm.draft}
            onChangeText={vm.onChangeDraft}
            onSend={vm.onSend}
            placeholder={vm.placeholder}
            sendLabel={vm.isSending ? '…' : vm.sendLabel}
            sendDisabled={vm.sendDisabled}
            disabled={vm.isSending}
          />
          <View style={styles.actions}>
            <Button
              label={vm.newChatLabel}
              variant="secondary"
              onPress={vm.onNewChat}
              disabled={vm.isSending}
            />
            {vm.showRetry ? (
              <Button
                label={vm.retryLabel}
                variant="secondary"
                onPress={vm.onRetryLast}
                disabled={vm.isSending}
              />
            ) : null}
          </View>
        </View>
      </View>
    </ScreenTemplate>
  );
}

function AiChatPageConnected() {
  const vm = useAiChatPageViewModel();
  return <AiChatPageView vm={vm} />;
}

export function AiChatPage({ viewModel }: AiChatPageProps = {}) {
  if (viewModel) {
    return <AiChatPageView vm={viewModel} />;
  }
  return <AiChatPageConnected />;
}
