import React, { useState, useEffect, useRef } from 'react';
import {
  View, Text, StyleSheet, FlatList,
  ActivityIndicator, TextInput, TouchableOpacity,
  Keyboard, Alert
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { tutoringService } from '../../../api/tutoring';
import { useKeyboardOverlap } from '../../../hooks/use-keyboard';
import { colors } from '../../../theme/colors';
import { typography } from '../../../theme/typography';

interface Message {
  id: string;
  type: 'AI' | 'USER';
  text: string;
}

export default function TutoringSessionScreen() {
  const { id, title } = useLocalSearchParams();
  const router = useRouter();

  const [sessionId, setSessionId] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputText, setInputText] = useState('');
  const [isTyping, setIsTyping] = useState(true);
  const flatListRef = useRef<FlatList>(null);

  // How much of the chat area the keyboard covers, on either platform.
  const { overlap, keyboardHeight, onLayout: measureChatArea } = useKeyboardOverlap();

  // Keep the newest message in view when the keyboard opens.
  useEffect(() => {
    if (keyboardHeight > 0) {
      const timer = setTimeout(() => flatListRef.current?.scrollToEnd({ animated: true }), 120);
      return () => clearTimeout(timer);
    }
  }, [keyboardHeight, overlap]);

  useEffect(() => {
    const initSession = async () => {
      try {
        const subject = typeof title === 'string' ? title : 'Study Document';
        const res: any = await tutoringService.startSession(subject, 'Comprehensive Review');

        if (res.data && res.data.sessionId) {
          setSessionId(res.data.sessionId);
          if (res.data.aiMessage) {
            setMessages([
              { id: Date.now().toString(), type: 'AI', text: res.data.aiMessage }
            ]);
          }
        }
      } catch (error: any) {
        console.error('Failed to start session', error);
        Alert.alert('Error', error.response?.data?.message || 'Failed to start session. Please try again.');
      } finally {
        setIsTyping(false);
      }
    };
    initSession();
  }, []);

  const handleSend = async () => {
    if (!inputText.trim() || !sessionId) return;

    const userMsgText = inputText.trim();
    const newUserMsg: Message = { id: Date.now().toString(), type: 'USER', text: userMsgText };
    setMessages(prev => [...prev, newUserMsg]);
    setInputText('');
    setIsTyping(true);
    Keyboard.dismiss();

    try {
      const res: any = await tutoringService.respondToTutor(sessionId, userMsgText);
      if (res.data && res.data.aiMessage) {
        const aiMsg: Message = { id: (Date.now() + 1).toString(), type: 'AI', text: res.data.aiMessage };
        setMessages(prev => [...prev, aiMsg]);
      }
    } catch (error) {
      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(),
        type: 'AI',
        text: 'Sorry, I ran into a problem. Could you try again?'
      }]);
    } finally {
      setIsTyping(false);
    }
  };

  const renderMessage = ({ item }: { item: Message }) => {
    const isUser = item.type === 'USER';
    return (
      <View style={[styles.messageRow, isUser && styles.messageRowUser]}>
        {!isUser && (
          <View style={styles.avatarAI}>
            <Text style={styles.avatarAIText}>🤖</Text>
          </View>
        )}
        <View style={[styles.bubble, isUser ? styles.bubbleUser : styles.bubbleAI]}>
          <Text style={[styles.bubbleText, isUser ? styles.bubbleTextUser : styles.bubbleTextAI]}>
            {item.text}
          </Text>
        </View>
      </View>
    );
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <View style={styles.headerMid}>
          <Text style={styles.headerTitle}>AI Tutor</Text>
          <Text style={styles.headerSubtitle} numberOfLines={1}>{title || 'Session'}</Text>
        </View>
        <View style={styles.headerDot}>
          <View style={styles.liveDot} />
          <Text style={styles.liveText}>Live</Text>
        </View>
      </View>

      {/* paddingBottom lifts the input bar clear of the keyboard */}
      <View style={[styles.flex, { paddingBottom: overlap }]} onLayout={measureChatArea}>
        <FlatList
          ref={flatListRef}
          data={messages}
          keyExtractor={item => item.id}
          renderItem={renderMessage}
          contentContainerStyle={styles.chatContent}
          onContentSizeChange={() => flatListRef.current?.scrollToEnd({ animated: true })}
          onLayout={() => flatListRef.current?.scrollToEnd({ animated: false })}
          showsVerticalScrollIndicator={false}
          keyboardShouldPersistTaps="handled"
          keyboardDismissMode="on-drag"
          ListEmptyComponent={
            !isTyping ? (
              <View style={styles.emptyChat}>
                <Text style={styles.emptyChatEmoji}>💬</Text>
                <Text style={styles.emptyChatText}>Your tutor is getting ready...</Text>
              </View>
            ) : null
          }
        />

        {/* Typing indicator */}
        {isTyping && (
          <View style={styles.typingRow}>
            <View style={styles.avatarAI}>
              <Text style={styles.avatarAIText}>🤖</Text>
            </View>
            <View style={styles.typingBubble}>
              <ActivityIndicator size="small" color={colors.primary} />
              <Text style={styles.typingText}>Thinking...</Text>
            </View>
          </View>
        )}

        {/* Input */}
        <View style={styles.inputBar}>
          <TextInput
            style={styles.input}
            placeholder="Type your answer..."
            placeholderTextColor={colors.textLight}
            value={inputText}
            onChangeText={setInputText}
            multiline
            maxLength={500}
          />
          <TouchableOpacity
            style={[styles.sendBtn, (!inputText.trim() || isTyping) && styles.sendBtnDisabled]}
            onPress={handleSend}
            disabled={!inputText.trim() || isTyping}
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
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  flex: { flex: 1 },

  // Header
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
  headerMid: {
    flex: 1,
    marginHorizontal: 12,
  },
  headerTitle: { ...typography.subtitle, fontSize: 16, color: colors.text },
  headerSubtitle: { ...typography.small, color: colors.textMuted },
  headerDot: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    backgroundColor: colors.successLight,
    borderRadius: 12,
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  liveDot: {
    width: 7, height: 7, borderRadius: 4,
    backgroundColor: colors.success,
  },
  liveText: { ...typography.small, color: colors.success, fontWeight: '700' },

  chatContent: {
    padding: 16,
    paddingBottom: 8,
    flexGrow: 1,
    justifyContent: 'flex-end',
  },

  // Messages
  messageRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    marginBottom: 12,
  },
  messageRowUser: {
    flexDirection: 'row-reverse',
  },
  avatarAI: {
    width: 34,
    height: 34,
    borderRadius: 17,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 8,
  },
  avatarAIText: { fontSize: 18 },
  bubble: {
    maxWidth: '80%',
    borderRadius: 20,
    padding: 14,
  },
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
  bubbleText: {
    ...typography.body,
    lineHeight: 22,
  },
  bubbleTextAI: { color: colors.text },
  bubbleTextUser: { color: '#FFFFFF' },

  // Typing
  typingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingBottom: 10,
  },
  typingBubble: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surfaceWarm,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    gap: 8,
    borderWidth: 1,
    borderColor: colors.border,
  },
  typingText: { ...typography.small, color: colors.textMuted },

  // Input bar
  inputBar: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: colors.surface,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    gap: 10,
  },
  input: {
    flex: 1,
    backgroundColor: colors.surfaceMid,
    borderWidth: 1.5,
    borderColor: colors.border,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    color: colors.text,
    ...typography.body,
    minHeight: 46,
    maxHeight: 120,
  },
  sendBtn: {
    width: 46,
    height: 46,
    borderRadius: 23,
    backgroundColor: colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
  },
  sendBtnDisabled: {
    backgroundColor: colors.border,
  },
  sendBtnText: {
    color: '#FFFFFF',
    fontSize: 22,
    fontWeight: '700',
    lineHeight: 28,
  },

  // Empty
  emptyChat: {
    alignItems: 'center',
    paddingVertical: 40,
  },
  emptyChatEmoji: { fontSize: 48, marginBottom: 12 },
  emptyChatText: { ...typography.body, color: colors.textMuted, textAlign: 'center' },
});
