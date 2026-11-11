import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, ScrollView, Alert } from 'react-native';
import { useLocalSearchParams, useRouter, useFocusEffect } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { mappingService } from '../../../../../api/mappings';
import { pyqService } from '../../../../../api/pyq';
import { quizService } from '../../../../../api/quiz';
import { masteryService } from '../../../../../api/mastery';
import { flashcardService } from '../../../../../api/flashcards';
import { colors } from '../../../../../theme/colors';
import { typography } from '../../../../../theme/typography';

const CATEGORY_EMOJI: Record<string, string> = { NOTES: '📝', PYQ: '🗞️', SYLLABUS: '📘', GENERAL: '📄' };
const FREQUENCY_COLOR: Record<string, string> = {
  HIGH: colors.error,
  MEDIUM: '#D97706',
  LOW: colors.textMuted,
  NONE: colors.textMuted,
};
const MAX_YEAR_GRID = 6;
const DIFFICULTIES: Array<'easy' | 'medium' | 'hard'> = ['easy', 'medium', 'hard'];
const MASTERY_COLOR: Record<string, string> = {
  STRONG: colors.success,
  DEVELOPING: '#D97706',
  WEAK: colors.error,
  NOT_STARTED: colors.textMuted,
};
const MASTERY_LABEL: Record<string, string> = {
  STRONG: 'Strong',
  DEVELOPING: 'Developing',
  WEAK: 'Weak',
  NOT_STARTED: 'Not started',
};

export default function TopicResourcesScreen() {
  const { id: examId, topicId } = useLocalSearchParams();
  const router = useRouter();

  const [data, setData] = useState<any>(null);
  const [pyqAnalysis, setPyqAnalysis] = useState<any>(null);
  const [quizzes, setQuizzes] = useState<any[]>([]);
  const [mastery, setMastery] = useState<any>(null);
  const [flashcards, setFlashcards] = useState<any[]>([]);
  const [generatingCards, setGeneratingCards] = useState(false);
  const [loading, setLoading] = useState(true);
  const [busyMappingId, setBusyMappingId] = useState<string | null>(null);
  const [busyQuestionId, setBusyQuestionId] = useState<string | null>(null);
  const [generatingQuiz, setGeneratingQuiz] = useState(false);
  const [quizDifficulty, setQuizDifficulty] = useState<'easy' | 'medium' | 'hard'>('medium');

  const load = async () => {
    try {
      const [resourcesRes, pyqRes, quizzesRes, masteryRes, flashcardsRes]: any[] = await Promise.all([
        mappingService.getTopicResources(examId as string, topicId as string),
        pyqService.getTopicPyqAnalysis(examId as string, topicId as string),
        quizService.listTopicQuizzes(examId as string, topicId as string),
        masteryService.getTopicMastery(examId as string, topicId as string),
        flashcardService.getTopicFlashcards(examId as string, topicId as string),
      ]);
      setData(resourcesRes.data);
      setPyqAnalysis(pyqRes.data);
      setQuizzes(quizzesRes.data);
      setMastery(masteryRes.data.mastery);
      setFlashcards(flashcardsRes.data);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to load this topic');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateFlashcards = async () => {
    try {
      setGeneratingCards(true);
      await flashcardService.generateTopicFlashcards(examId as string, topicId as string, { numberOfCards: 10 });
      await load();
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to generate flashcards for this topic');
    } finally {
      setGeneratingCards(false);
    }
  };

  const handleGenerateQuiz = async () => {
    try {
      setGeneratingQuiz(true);
      const res: any = await quizService.generateTopicQuiz(examId as string, topicId as string, {
        difficulty: quizDifficulty,
        numberOfQuestions: 5,
      });
      router.push({ pathname: '/(app)/quiz/[id]', params: { id: res.data.id } } as any);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to generate a quiz for this topic');
    } finally {
      setGeneratingQuiz(false);
    }
  };

  useFocusEffect(
    useCallback(() => {
      if (examId && topicId) load();
    }, [examId, topicId])
  );

  const handleReview = async (mappingId: string, status: 'CONFIRMED' | 'REJECTED') => {
    try {
      setBusyMappingId(mappingId);
      await mappingService.updateMappingStatus(mappingId, status);
      await load();
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to update this mapping');
    } finally {
      setBusyMappingId(null);
    }
  };

  const handleQuestionReview = async (questionId: string, status: 'CONFIRMED' | 'REJECTED') => {
    try {
      setBusyQuestionId(questionId);
      await pyqService.updateQuestionStatus(questionId, status);
      await load();
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to update this question');
    } finally {
      setBusyQuestionId(null);
    }
  };

  const formatPages = (start: number | null, end: number | null) => {
    if (!start && !end) return null;
    if (start && end && start !== end) return `Pages ${start}–${end}`;
    return `Page ${start || end}`;
  };

  const renderMapping = (m: any) => {
    const pages = formatPages(m.startPage, m.endPage);
    const isPending = m.status === 'PENDING_REVIEW';
    const isBusy = busyMappingId === m.id;

    return (
      <TouchableOpacity
        key={m.id}
        style={[styles.card, isPending && styles.cardPending]}
        onPress={() => router.push({ pathname: '/(app)/documents/[id]', params: { id: m.document.id } } as any)}
        activeOpacity={0.85}
      >
        <View style={styles.cardTopRow}>
          <Text style={styles.cardEmoji}>{CATEGORY_EMOJI[m.document.category] || '📄'}</Text>
          <View style={{ flex: 1 }}>
            <Text style={styles.cardTitle} numberOfLines={1}>{m.document.title}</Text>
            <Text style={styles.cardMeta}>
              {pages || 'No page range'}{m.subtopic ? ` · ${m.subtopic.name}` : ''}
            </Text>
          </View>
          {isPending && (
            <View style={styles.pendingBadge}>
              <Text style={styles.pendingBadgeText}>NEEDS REVIEW</Text>
            </View>
          )}
        </View>

        {isPending && (
          <View style={styles.reviewActions}>
            <TouchableOpacity
              style={[styles.reviewBtn, styles.rejectBtn]}
              disabled={isBusy}
              onPress={() => handleReview(m.id, 'REJECTED')}
            >
              <Text style={styles.rejectBtnText}>{isBusy ? '...' : 'Reject'}</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[styles.reviewBtn, styles.confirmBtn]}
              disabled={isBusy}
              onPress={() => handleReview(m.id, 'CONFIRMED')}
            >
              <Text style={styles.confirmBtnText}>{isBusy ? '...' : 'Confirm'}</Text>
            </TouchableOpacity>
          </View>
        )}
      </TouchableOpacity>
    );
  };

  if (loading || !data) {
    return (
      <SafeAreaView style={styles.loadingContainer} edges={['top', 'bottom']}>
        <ActivityIndicator size="large" color={colors.primary} />
      </SafeAreaView>
    );
  }

  const freq = pyqAnalysis?.frequency;
  const questions = pyqAnalysis?.questions || [];
  const hasAnything = data.notes.length > 0 || data.pyqs.length > 0 || data.syllabus.length > 0 || questions.length > 0;

  // A recent window of years, checked off against which ones this topic
  // actually appeared in — the ✓/✗ grid from the spec's example.
  const yearGrid = (() => {
    if (!freq || freq.yearsSeen.length === 0) return [];
    const currentYear = new Date().getFullYear();
    const maxYear = Math.max(currentYear, ...freq.yearsSeen);
    const minYear = Math.max(Math.min(...freq.yearsSeen), maxYear - MAX_YEAR_GRID + 1);
    const years: number[] = [];
    for (let y = maxYear; y >= minYear; y--) years.push(y);
    return years;
  })();

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle} numberOfLines={1}>{data.topic.name}</Text>
        <View style={{ width: 36 }} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        {freq && freq.level !== 'NONE' && (
          <View style={styles.freqCard}>
            <View style={styles.freqHeaderRow}>
              <Text style={styles.freqTitle}>📊 Historical Importance</Text>
              <View style={[styles.freqBadge, { backgroundColor: FREQUENCY_COLOR[freq.level] + '20' }]}>
                <Text style={[styles.freqBadgeText, { color: FREQUENCY_COLOR[freq.level] }]}>{freq.level}</Text>
              </View>
            </View>
            <Text style={styles.freqSubtext}>
              Appeared in {freq.yearsSeen.length} of the last {yearGrid.length || freq.yearsSeen.length} years you&apos;ve
              uploaded papers for — a relative signal from your own materials, not a prediction.
            </Text>
            {yearGrid.length > 0 && (
              <View style={styles.yearGrid}>
                {yearGrid.map((y) => {
                  const seen = freq.yearsSeen.includes(y);
                  return (
                    <View key={y} style={[styles.yearChip, seen ? styles.yearChipSeen : styles.yearChipMissed]}>
                      <Text style={styles.yearChipLabel}>{y}</Text>
                      <Text style={[styles.yearChipMark, { color: seen ? colors.success : colors.textLight }]}>
                        {seen ? '✓' : '✗'}
                      </Text>
                    </View>
                  );
                })}
              </View>
            )}
          </View>
        )}

        {mastery && (
          <View style={styles.freqCard}>
            <View style={styles.freqHeaderRow}>
              <Text style={styles.freqTitle}>🎯 Your Mastery</Text>
              <View style={[styles.freqBadge, { backgroundColor: MASTERY_COLOR[mastery.level] + '20' }]}>
                <Text style={[styles.freqBadgeText, { color: MASTERY_COLOR[mastery.level] }]}>
                  {MASTERY_LABEL[mastery.level]}
                </Text>
              </View>
            </View>
            <Text style={styles.freqSubtext}>
              {mastery.attemptCount === 0
                ? "You haven't attempted any quiz questions on this topic yet."
                : `${Math.round(mastery.accuracy * 100)}% correct across ${mastery.attemptCount} question${mastery.attemptCount === 1 ? '' : 's'} answered, weighted toward your more recent attempts.${mastery.lowConfidence ? ' A few more attempts will make this more reliable.' : ''}`}
            </Text>
          </View>
        )}

        <View style={styles.section}>
          <Text style={styles.sectionTitle}>🧠 Practice Quiz</Text>

          <View style={styles.difficultyRow}>
            {DIFFICULTIES.map((d) => (
              <TouchableOpacity
                key={d}
                style={[styles.difficultyChip, quizDifficulty === d && styles.difficultyChipActive]}
                onPress={() => setQuizDifficulty(d)}
              >
                <Text style={[styles.difficultyChipText, quizDifficulty === d && styles.difficultyChipTextActive]}>
                  {d}
                </Text>
              </TouchableOpacity>
            ))}
          </View>

          <TouchableOpacity
            style={styles.generateQuizBtn}
            onPress={handleGenerateQuiz}
            disabled={generatingQuiz}
            activeOpacity={0.85}
          >
            {generatingQuiz ? (
              <ActivityIndicator color="#FFFFFF" />
            ) : (
              <Text style={styles.generateQuizBtnText}>+ Generate New Quiz</Text>
            )}
          </TouchableOpacity>

          {quizzes.length > 0 && quizzes.map((q: any) => (
            <TouchableOpacity
              key={q.id}
              style={styles.card}
              onPress={() => router.push({ pathname: '/(app)/quiz/[id]', params: { id: q.id } } as any)}
              activeOpacity={0.85}
            >
              <View style={styles.cardTopRow}>
                <Text style={styles.cardEmoji}>📝</Text>
                <View style={{ flex: 1 }}>
                  <Text style={styles.cardTitle} numberOfLines={1}>
                    {q.questionCount} questions · {q.difficulty}
                  </Text>
                  <Text style={styles.cardMeta}>
                    {q.lastAttempt ? `Last score: ${Math.round(q.lastAttempt.score)}%` : 'Not attempted yet'}
                  </Text>
                </View>
              </View>
            </TouchableOpacity>
          ))}
        </View>

        <View style={styles.section}>
          <Text style={styles.sectionTitle}>🔁 Flashcards</Text>
          <Text style={styles.flashcardStats}>
            {flashcards.length === 0
              ? 'No flashcards yet for this topic.'
              : `${flashcards.length} card${flashcards.length === 1 ? '' : 's'} · ${flashcards.filter((c) => new Date(c.dueAt) <= new Date()).length} due now`}
          </Text>
          <TouchableOpacity
            style={styles.generateQuizBtn}
            onPress={handleGenerateFlashcards}
            disabled={generatingCards}
            activeOpacity={0.85}
          >
            {generatingCards ? (
              <ActivityIndicator color="#FFFFFF" />
            ) : (
              <Text style={styles.generateQuizBtnText}>+ Generate Flashcards</Text>
            )}
          </TouchableOpacity>
        </View>

        {questions.length > 0 && (
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>🗞️ Past Year Questions ({questions.length})</Text>
            {questions.map((q: any) => {
              const isPending = q.status === 'PENDING_REVIEW';
              const isBusy = busyQuestionId === q.id;
              return (
                <View key={q.id} style={[styles.questionCard, isPending && styles.cardPending]}>
                  <View style={styles.questionTopRow}>
                    <Text style={styles.questionText}>{q.questionText}</Text>
                    {q.year && (
                      <View style={styles.yearTag}>
                        <Text style={styles.yearTagText}>{q.year}</Text>
                      </View>
                    )}
                  </View>
                  {q.correctAnswer ? <Text style={styles.answerText}>✓ {q.correctAnswer}</Text> : null}
                  <TouchableOpacity onPress={() => router.push({ pathname: '/(app)/documents/[id]', params: { id: q.document.id } } as any)}>
                    <Text style={styles.questionSourceLink} numberOfLines={1}>From: {q.document.title}</Text>
                  </TouchableOpacity>

                  {isPending && (
                    <View style={styles.reviewActions}>
                      <TouchableOpacity
                        style={[styles.reviewBtn, styles.rejectBtn]}
                        disabled={isBusy}
                        onPress={() => handleQuestionReview(q.id, 'REJECTED')}
                      >
                        <Text style={styles.rejectBtnText}>{isBusy ? '...' : 'Reject'}</Text>
                      </TouchableOpacity>
                      <TouchableOpacity
                        style={[styles.reviewBtn, styles.confirmBtn]}
                        disabled={isBusy}
                        onPress={() => handleQuestionReview(q.id, 'CONFIRMED')}
                      >
                        <Text style={styles.confirmBtnText}>{isBusy ? '...' : 'Confirm'}</Text>
                      </TouchableOpacity>
                    </View>
                  )}
                </View>
              );
            })}
          </View>
        )}

        {!hasAnything ? (
          <View style={styles.emptyState}>
            <Text style={styles.emptyEmoji}>🔍</Text>
            <Text style={styles.emptyText}>
              No resources are mapped to this topic yet. Upload notes or past papers linked to this exam, and they&apos;ll
              be mapped automatically.
            </Text>
          </View>
        ) : (
          <>
            {data.notes.length > 0 && (
              <View style={styles.section}>
                <Text style={styles.sectionTitle}>📝 Notes</Text>
                {data.notes.map(renderMapping)}
              </View>
            )}
            {data.pyqs.length > 0 && (
              <View style={styles.section}>
                <Text style={styles.sectionTitle}>🗞️ Past Paper Resources</Text>
                {data.pyqs.map(renderMapping)}
              </View>
            )}
            {data.syllabus.length > 0 && (
              <View style={styles.section}>
                <Text style={styles.sectionTitle}>📘 Syllabus References</Text>
                {data.syllabus.map(renderMapping)}
              </View>
            )}
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1, backgroundColor: colors.background },
  loadingContainer: { flex: 1, backgroundColor: colors.background, justifyContent: 'center', alignItems: 'center' },

  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingVertical: 14,
  },
  headerBack: {
    width: 36, height: 36, borderRadius: 18,
    backgroundColor: colors.surfaceWarm,
    justifyContent: 'center', alignItems: 'center',
  },
  headerBackText: { fontSize: 24, color: colors.text, lineHeight: 28 },
  headerTitle: { ...typography.subtitle, color: colors.text, flex: 1, textAlign: 'center', marginHorizontal: 8 },

  scrollContent: {
    paddingHorizontal: 20,
    paddingBottom: 40,
    width: '100%',
    maxWidth: 520,
    alignSelf: 'center',
  },

  section: { marginBottom: 24 },
  sectionTitle: { ...typography.subtitle, fontSize: 16, color: colors.text, marginBottom: 12 },

  card: {
    backgroundColor: colors.surface,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 14,
    marginBottom: 10,
  },
  cardPending: {
    borderColor: '#F0A742' + '60',
    backgroundColor: '#FEF3C7' + '30',
  },
  cardTopRow: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  cardEmoji: { fontSize: 22 },
  cardTitle: { ...typography.body, fontWeight: '600', color: colors.text },
  cardMeta: { ...typography.small, color: colors.textMuted, marginTop: 2 },

  pendingBadge: {
    backgroundColor: '#FEF3C7',
    borderRadius: 10,
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  pendingBadgeText: { ...typography.small, fontSize: 9, fontWeight: '700', color: '#92640A' },

  reviewActions: { flexDirection: 'row', gap: 8, marginTop: 12 },
  reviewBtn: { flex: 1, paddingVertical: 8, borderRadius: 10, alignItems: 'center' },
  rejectBtn: { backgroundColor: colors.surfaceWarm, borderWidth: 1, borderColor: colors.border },
  rejectBtnText: { ...typography.caption, fontWeight: '700', color: colors.textMuted },
  confirmBtn: { backgroundColor: colors.success },
  confirmBtnText: { ...typography.caption, fontWeight: '700', color: '#FFFFFF' },

  emptyState: { alignItems: 'center', paddingTop: 60, paddingHorizontal: 20 },
  emptyEmoji: { fontSize: 48, marginBottom: 16 },
  emptyText: { ...typography.body, color: colors.textMuted, textAlign: 'center', lineHeight: 22 },

  freqCard: {
    backgroundColor: colors.surface,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 16,
    marginBottom: 24,
  },
  freqHeaderRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  freqTitle: { ...typography.subtitle, fontSize: 15, color: colors.text },
  freqBadge: { borderRadius: 10, paddingHorizontal: 10, paddingVertical: 4 },
  freqBadgeText: { ...typography.small, fontWeight: '800' },
  freqSubtext: { ...typography.small, color: colors.textMuted, marginTop: 8, lineHeight: 17 },

  yearGrid: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginTop: 14 },
  yearChip: {
    alignItems: 'center',
    borderRadius: 12,
    paddingVertical: 8,
    paddingHorizontal: 10,
    minWidth: 52,
    borderWidth: 1,
  },
  yearChipSeen: { backgroundColor: colors.successLight, borderColor: colors.success + '40' },
  yearChipMissed: { backgroundColor: colors.surfaceWarm, borderColor: colors.border },
  yearChipLabel: { ...typography.small, fontWeight: '700', color: colors.text },
  yearChipMark: { ...typography.small, fontWeight: '800', marginTop: 2 },

  difficultyRow: { flexDirection: 'row', gap: 8, marginBottom: 12 },
  difficultyChip: {
    flex: 1,
    alignItems: 'center',
    paddingVertical: 8,
    borderRadius: 10,
    backgroundColor: colors.surfaceWarm,
    borderWidth: 1,
    borderColor: colors.border,
  },
  difficultyChipActive: { backgroundColor: colors.primary, borderColor: colors.primary },
  difficultyChipText: { ...typography.small, fontWeight: '700', color: colors.textMuted, textTransform: 'capitalize' },
  difficultyChipTextActive: { color: '#FFFFFF' },
  generateQuizBtn: {
    backgroundColor: colors.primary,
    borderRadius: 14,
    paddingVertical: 14,
    alignItems: 'center',
    marginBottom: 12,
  },
  generateQuizBtnText: { ...typography.body, fontWeight: '700', color: '#FFFFFF' },
  flashcardStats: { ...typography.small, color: colors.textMuted, marginBottom: 12 },

  questionCard: {
    backgroundColor: colors.surface,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 14,
    marginBottom: 10,
  },
  questionTopRow: { flexDirection: 'row', justifyContent: 'space-between', gap: 8 },
  questionText: { ...typography.body, color: colors.text, lineHeight: 21, flex: 1 },
  yearTag: { backgroundColor: colors.surfaceWarm, borderRadius: 8, paddingHorizontal: 8, paddingVertical: 3, height: 22 },
  yearTagText: { ...typography.small, fontWeight: '700', color: colors.textMuted },
  answerText: { ...typography.small, color: colors.success, fontWeight: '700', marginTop: 8 },
  questionSourceLink: { ...typography.small, color: colors.primary, marginTop: 8, fontWeight: '600' },
});
