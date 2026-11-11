import React, { useState, useEffect, useRef } from 'react';
import {
  View, Text, StyleSheet, FlatList,
  ActivityIndicator, TextInput, TouchableOpacity,
  Keyboard,
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { aiService } from '../../api/ai';
import { useKeyboardOverlap } from '../../hooks/use-keyboard';
import { colors } from '../../theme/colors';
import { typography } from '../../theme/typography';

interface Source {
  title: string;
  page: number | null;
}

interface Message {
  id: string;
  type: 'AI' | 'USER';
  text: string;
  sources?: Source[];
  foundInMaterials?: boolean;
  isError?: boolean;
}

export default function AskScreen() {
  const { examId, examName } = useLocalSearchParams();
  const scopedExamId = typeof examId === 'string' ? examId : undefined;
  const router = useRouter();

  const [messages, setMessages] = useState<Message[]>([
    {
      id: 'intro',
      type: 'AI',
      text: scopedExamId
        ? `Ask me anything about your resources for this exam — I'll answer from what you've uploaded and tell you clearly when something isn't in there.`
        : `Ask me anything — I'll answer from your uploaded study materials and tell you clearly when something isn't in there.`,
    },
  ]);
  const [inputText, setInputText] = useState('');
  const [isThinking, setIsThinking] = useState(false);
  const flatListRef = useRef<FlatList>(null);

  const { overlap, keyboardHeight, onLayout: measureChatArea } = useKeyboardOverlap();

  useEffect(() => {
    if (keyboardHeight > 0) {
      const timer = setTimeout(() => flatListRef.current?.scrollToEnd({ animated: true }), 120);
      return () => clearTimeout(timer);
    }
  }, [keyboardHeight, overlap]);

  const handleSend = async () => {
    if (!inputText.trim() || isThinking) return;

    const question = inputText.trim();
    setMessages((prev) => [...prev, { id: `${Date.now()}`, type: 'USER', text: question }]);
    setInputText('');
    setIsThinking(true);
    Keyboard.dismiss();

    try {
      const res: any = await aiService.askQuestion(question, { examId: scopedExamId });
      setMessages((prev) => [
        ...prev,
        {
          id: `${Date.now()}-ai`,
          type: 'AI',
          text: res.data.answer,
          sources: res.data.sources,
          foundInMaterials: res.data.foundInMaterials,
        },
      ]);
    } catch (error: any) {
      setMessages((prev) => [
        ...prev,
        {
          id: `${Date.now()}-err`,
          type: 'AI',
          text: error.response?.data?.message || "Sorry, I ran into a problem answering that. Please try again.",
          isError: true,
        },
      ]);
    } finally {
      setIsThinking(false);
    }
  };

  const formatSource = (s: Source) => (s.page ? `${s.title}, Page ${s.page}` : s.title);

  const renderMessage = ({ item }: { item: Message }) => {
    const isUser = item.type === 'USER';
    return (
      <View style={[styles.messageRow, isUser && styles.messageRowUser]}>
        {!isUser && (
          <View style={styles.avatarAI}>
            <Text style={styles.avatarAIText}>✨</Text>
          </View>
        )}
        <View style={[styles.bubble, isUser ? styles.bubbleUser : styles.bubbleAI, item.isError && styles.bubbleError]}>
          <Text style={[styles.bubbleText, isUser ? styles.bubbleTextUser : styles.bubbleTextAI]}>
            {item.text}
          </Text>

          {!isUser && item.foundInMaterials === false && !item.isError && (
            <View style={styles.notFoundPill}>
              <Text style={styles.notFoundPillText}>Not found in your uploaded materials</Text>
            </View>
          )}

          {!isUser && item.sources && item.sources.length > 0 && (
            <View style={styles.sourcesBox}>
              <Text style={styles.sourcesLabel}>SOURCES</Text>
              {item.sources.map((s, i) => (
                <Text key={i} style={styles.sourceText}>📄 {formatSource(s)}</Text>
              ))}
            </View>
          )}
        </View>
      </View>
    );
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <View style={styles.headerMid}>
          <Text style={styles.headerTitle}>Ask AI</Text>
          <Text style={styles.headerSubtitle} numberOfLines={1}>
            {scopedExamId ? `Scoped to ${typeof examName === 'string' ? examName : 'this exam'}` : 'All your materials'}
          </Text>
        </View>
        <View style={{ width: 36 }} />
      </View>

      <View style={[styles.flex, { paddingBottom: overlap }]} onLayout={measureChatArea}>
        <FlatList
          ref={flatListRef}
          data={messages}
          keyExtractor={(item) => item.id}
          renderItem={renderMessage}
          contentContainerStyle={styles.chatContent}
          onContentSizeChange={() => flatListRef.current?.scrollToEnd({ animated: true })}
          showsVerticalScrollIndicator={false}
          keyboardShouldPersistTaps="handled"
          keyboardDismissMode="on-drag"
        />

        {isThinking && (
          <View style={styles.typingRow}>
            <View style={styles.avatarAI}>
              <Text style={styles.avatarAIText}>✨</Text>
            </View>
            <View style={styles.typingBubble}>
              <ActivityIndicator size="small" color={colors.primary} />
              <Text style={styles.typingText}>Searching your materials...</Text>
            </View>
          </View>
        )}

        <View style={styles.inputBar}>
          <TextInput
            style={styles.input}
            placeholder="e.g. Explain deadlocks using my notes"
            placeholderTextColor={colors.textLight}
            value={inputText}
            onChangeText={setInputText}
            multiline
            maxLength={500}
          />
          <TouchableOpacity
            style={[styles.sendBtn, (!inputText.trim() || isThinking) && styles.sendBtnDisabled]}
            onPress={handleSend}
            disabled={!inputText.trim() || isThinking}
            activeOpacity={0.8}
          >
            <Text style={styles.sendBtnText}>↑</Text>
          </TouchableOpacity>
        </View>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1, backgroundColor: colors.background },
  flex: { flex: 1 },

  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: colors.surface,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  headerBack: {
    width: 36, height: 36, borderRadius: 18,
    backgroundColor: colors.surfaceWarm,
    justifyContent: 'center', alignItems: 'center',
  },
  headerBackText: { fontSize: 24, color: colors.text, lineHeight: 28 },
  headerMid: { flex: 1, marginHorizontal: 12 },
  headerTitle: { ...typography.subtitle, fontSize: 16, color: colors.text },
  headerSubtitle: { ...typography.small, color: colors.textMuted },

  chatContent: { padding: 16, paddingBottom: 8, flexGrow: 1, justifyContent: 'flex-end' },

  messageRow: { flexDirection: 'row', alignItems: 'flex-end', marginBottom: 12 },
  messageRowUser: { flexDirection: 'row-reverse' },
  avatarAI: {
    width: 34, height: 34, borderRadius: 17,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center', alignItems: 'center',
    marginRight: 8,
  },
  avatarAIText: { fontSize: 18 },
  bubble: { maxWidth: '82%', borderRadius: 20, padding: 14 },
  bubbleAI: {
    backgroundColor: colors.surfaceWarm,
    borderBottomLeftRadius: 4,
    borderWidth: 1,
    borderColor: colors.border,
  },
  bubbleUser: {
    backgroundColor: colors.success,
    borderBottomRightRadius: 4,
    marginLeft: 8,
  },
  bubbleError: {
    backgroundColor: '#FEE2E2',
    borderColor: colors.error + '40',
  },
  bubbleText: { ...typography.body, lineHeight: 22 },
  bubbleTextAI: { color: colors.text },
  bubbleTextUser: { color: '#FFFFFF' },

  notFoundPill: {
    marginTop: 10,
    alignSelf: 'flex-start',
    backgroundColor: '#FEF3C7',
    borderRadius: 12,
    paddingHorizontal: 10,
    paddingVertical: 5,
  },
  notFoundPillText: { ...typography.small, color: '#92640A', fontWeight: '700' },

  sourcesBox: {
    marginTop: 12,
    paddingTop: 10,
    borderTopWidth: 1,
    borderTopColor: colors.divider,
  },
  sourcesLabel: { ...typography.label, fontSize: 10, color: colors.textMuted, marginBottom: 6 },
  sourceText: { ...typography.small, color: colors.textMuted, marginTop: 2 },

  typingRow: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: 16, paddingBottom: 10 },
  typingBubble: {
    flexDirection: 'row', alignItems: 'center',
    backgroundColor: colors.surfaceWarm,
    borderRadius: 20, paddingHorizontal: 16, paddingVertical: 10, gap: 8,
    borderWidth: 1, borderColor: colors.border,
  },
  typingText: { ...typography.small, color: colors.textMuted },

  inputBar: {
    flexDirection: 'row', alignItems: 'flex-end',
    paddingHorizontal: 16, paddingVertical: 12,
    backgroundColor: colors.surface,
    borderTopWidth: 1, borderTopColor: colors.border,
    gap: 10,
  },
  input: {
    flex: 1,
    backgroundColor: colors.surfaceMid,
    borderWidth: 1.5, borderColor: colors.border,
    borderRadius: 20, paddingHorizontal: 16, paddingVertical: 10,
    color: colors.text,
    ...typography.body,
    minHeight: 46, maxHeight: 120,
  },
  sendBtn: {
    width: 46, height: 46, borderRadius: 23,
    backgroundColor: colors.primary,
    justifyContent: 'center', alignItems: 'center',
  },
  sendBtnDisabled: { backgroundColor: colors.border },
  sendBtnText: { color: '#FFFFFF', fontSize: 22, fontWeight: '700', lineHeight: 28 },
});
