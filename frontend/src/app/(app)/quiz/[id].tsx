import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
  ScrollView,
  Alert,
  Animated
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { quizService } from '../../../api/quiz';
import { colors } from '../../../theme/colors';
import { typography } from '../../../theme/typography';

export default function QuizScreen() {
  const { id: quizId } = useLocalSearchParams();
  const router = useRouter();

  const [quiz, setQuiz] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [selectedAnswers, setSelectedAnswers] = useState<Record<string, string>>({});
  const [result, setResult] = useState<any>(null);
  const [answerScale] = useState(new Animated.Value(1));

  useEffect(() => {
    const fetchQuiz = async () => {
      try {
        const res: any = await quizService.getQuiz(quizId as string);
        setQuiz(res.data);
      } catch (error: any) {
        Alert.alert('Error', error.response?.data?.message || 'Failed to load this quiz.');
        router.back();
      } finally {
        setLoading(false);
      }
    };
    if (quizId) fetchQuiz();
  }, [quizId]);

  if (loading) {
    return (
      <View style={styles.loadingContainer}>
        <View style={styles.loadingCard}>
          <Text style={styles.loadingEmoji}>🧠</Text>
          <ActivityIndicator size="large" color={colors.primary} style={{ marginVertical: 16 }} />
          <Text style={styles.loadingTitle}>Loading Your Quiz...</Text>
        </View>
      </View>
    );
  }

  if (!quiz || !quiz.questions || quiz.questions.length === 0) {
    return (
      <View style={styles.loadingContainer}>
        <Text style={styles.errorEmoji}>😕</Text>
        <Text style={styles.errorText}>Could not load this quiz</Text>
        <TouchableOpacity style={styles.backBtn} onPress={() => router.back()}>
          <Text style={styles.backBtnText}>← Go Back</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const questions = quiz.questions;
  const currentQuestion = questions[currentQuestionIndex];

  const handleSelectOption = (option: string) => {
    if (selectedAnswers[currentQuestion.id]) return;

    Animated.sequence([
      Animated.timing(answerScale, { toValue: 0.97, duration: 80, useNativeDriver: true }),
      Animated.spring(answerScale, { toValue: 1, useNativeDriver: true, speed: 30, bounciness: 6 }),
    ]).start();

    setSelectedAnswers(prev => ({ ...prev, [currentQuestion.id]: option }));
  };

  const handleNext = async () => {
    if (currentQuestionIndex < questions.length - 1) {
      setCurrentQuestionIndex(prev => prev + 1);
      return;
    }

    // Last question answered — grade server-side. Scoring is never computed
    // on the client; the backend is the sole source of truth for correctness.
    try {
      setSubmitting(true);
      const answers = questions.map((q: any) => ({
        questionId: q.id,
        selectedAnswer: selectedAnswers[q.id] ?? null,
      }));
      const res: any = await quizService.submitQuizAttempt(quizId as string, answers);
      setResult(res.data);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to submit your answers.');
    } finally {
      setSubmitting(false);
    }
  };

  if (result) {
    const percentage = Math.round(result.score);
    const { emoji, message } = percentage >= 80
      ? { emoji: '🌟', message: 'Outstanding! You really know this!' }
      : percentage >= 50
        ? { emoji: '👍', message: 'Good effort! Keep studying!' }
        : { emoji: '📖', message: 'Review the material and try again!' };

    return (
      <SafeAreaView style={styles.safeArea}>
        <ScrollView
          contentContainerStyle={styles.resultsContainer}
          showsVerticalScrollIndicator={false}
        >
          <Text style={styles.resultsEmoji}>{emoji}</Text>
          <Text style={styles.resultsTitle}>Quiz Complete!</Text>

          <View style={styles.scoreCard}>
            <View style={[styles.scoreCircle, { borderColor: percentage >= 50 ? colors.success : colors.error }]}>
              <Text style={[styles.scoreNumber, { color: percentage >= 50 ? colors.success : colors.error }]}>
                {result.correctCount}/{result.totalQuestions}
              </Text>
              <Text style={styles.scorePercent}>{percentage}%</Text>
            </View>
          </View>

          <Text style={styles.resultsMessage}>{message}</Text>

          <View style={styles.reviewSection}>
            <Text style={styles.reviewTitle}>Review Answers</Text>
            {result.answers.map((a: any, idx: number) => (
              <View key={a.questionId} style={styles.reviewCard}>
                <Text style={styles.reviewQuestion}>{idx + 1}. {a.question}</Text>
                <Text style={[styles.reviewAnswer, { color: a.isCorrect ? colors.success : colors.error }]}>
                  {a.isCorrect ? '✓' : '✗'} Your answer: {a.selectedAnswer || '(skipped)'}
                </Text>
                {!a.isCorrect && (
                  <Text style={styles.reviewCorrect}>Correct answer: {a.correctAnswer}</Text>
                )}
                {a.explanation && <Text style={styles.reviewExplanation}>{a.explanation}</Text>}
              </View>
            ))}
          </View>

          <TouchableOpacity style={styles.doneBtn} onPress={() => router.back()} activeOpacity={0.85}>
            <Text style={styles.doneBtnText}>Back to Topic →</Text>
          </TouchableOpacity>
        </ScrollView>
      </SafeAreaView>
    );
  }

  const selectedForCurrent = selectedAnswers[currentQuestion.id];
  const progress = (currentQuestionIndex + 1) / questions.length;
  const isLast = currentQuestionIndex === questions.length - 1;

  return (
    <SafeAreaView style={styles.safeArea}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle} numberOfLines={1}>{quiz.title || 'Quiz'}</Text>
        <Text style={styles.headerProgress}>
          <Text style={styles.headerProgressCurrent}>{currentQuestionIndex + 1}</Text>/{questions.length}
        </Text>
      </View>

      {/* Progress Bar */}
      <View style={styles.progressBarBg}>
        <View style={[styles.progressBarFill, { width: `${progress * 100}%` as any }]} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        {/* Question Card */}
        <View style={styles.questionCard}>
          <Text style={styles.questionLabel}>Question {currentQuestionIndex + 1}</Text>
          <Text style={styles.questionText}>{currentQuestion.question}</Text>
        </View>

        {/* Options */}
        <View style={styles.optionsContainer}>
          {currentQuestion.options.map((option: string, index: number) => {
            const isSelected = selectedForCurrent === option;

            return (
              <Animated.View key={index} style={{ transform: [{ scale: isSelected ? answerScale : new Animated.Value(1) }] }}>
                <TouchableOpacity
                  style={[styles.optionBtn, isSelected ? styles.optionSelected : styles.optionDefault]}
                  onPress={() => handleSelectOption(option)}
                  activeOpacity={selectedForCurrent ? 1 : 0.8}
                >
                  <Text style={[styles.optionText, styles.optionTextDefault]}>{option}</Text>
                  {isSelected && <Text style={styles.optionArrow}>✓</Text>}
                </TouchableOpacity>
              </Animated.View>
            );
          })}
        </View>
      </ScrollView>

      {/* Footer */}
      {selectedForCurrent && (
        <View style={styles.footer}>
          <TouchableOpacity style={styles.nextBtn} onPress={handleNext} activeOpacity={0.85} disabled={submitting}>
            {submitting ? (
              <ActivityIndicator color="#FFFFFF" />
            ) : (
              <Text style={styles.nextBtnText}>
                {isLast ? 'Submit Quiz 🎉' : 'Next Question →'}
              </Text>
            )}
          </TouchableOpacity>
        </View>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  loadingContainer: {
    flex: 1,
    backgroundColor: colors.background,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 40,
  },
  loadingCard: {
    backgroundColor: colors.surface,
    borderRadius: 24,
    padding: 32,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: colors.border,
    width: '100%',
  },
  loadingEmoji: { fontSize: 56, marginBottom: 4 },
  loadingTitle: { ...typography.subtitle, color: colors.text },
  loadingSubtitle: { ...typography.body, color: colors.textMuted, textAlign: 'center', marginTop: 4 },
  errorEmoji: { fontSize: 56, marginBottom: 12 },
  errorText: { ...typography.subtitle, color: colors.text, marginBottom: 24 },
  backBtn: {
    backgroundColor: colors.surfaceWarm,
    borderRadius: 16,
    paddingHorizontal: 24,
    paddingVertical: 12,
  },
  backBtnText: { ...typography.body, fontWeight: '700', color: colors.text },

  // Header
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingVertical: 14,
  },
  headerBack: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: colors.surfaceWarm,
    justifyContent: 'center',
    alignItems: 'center',
  },
  headerBackText: { fontSize: 24, color: colors.text, lineHeight: 28 },
  headerTitle: { ...typography.subtitle, color: colors.text, flex: 1, textAlign: 'center', marginHorizontal: 8 },
  headerProgress: { ...typography.body, color: colors.textMuted, fontWeight: '600', minWidth: 36, textAlign: 'right' },
  headerProgressCurrent: { color: colors.success, fontWeight: '800' },

  // Progress bar
  progressBarBg: {
    height: 6,
    backgroundColor: colors.border,
    marginHorizontal: 20,
    borderRadius: 3,
    marginBottom: 20,
    overflow: 'hidden',
  },
  progressBarFill: {
    height: '100%',
    backgroundColor: colors.success,
    borderRadius: 3,
  },

  // Scroll content
  scrollContent: {
    paddingHorizontal: 20,
    paddingBottom: 40,
  },

  // Question
  questionCard: {
    backgroundColor: colors.surfaceWarm,
    borderRadius: 20,
    padding: 24,
    marginBottom: 20,
    borderWidth: 1,
    borderColor: colors.border,
  },
  questionLabel: {
    ...typography.label,
    color: colors.primary,
    marginBottom: 10,
  },
  questionText: {
    ...typography.title,
    fontSize: 21,
    color: colors.text,
    lineHeight: 30,
  },

  // Options
  optionsContainer: { gap: 12 },
  optionBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    borderRadius: 16,
    paddingVertical: 18,
    paddingHorizontal: 20,
  },
  optionDefault: {
    backgroundColor: colors.primary,
  },
  optionSelected: {
    backgroundColor: colors.success,
  },
  optionText: {
    ...typography.body,
    fontWeight: '600',
    flex: 1,
  },
  optionTextDefault: { color: '#FFFFFF' },
  optionArrow: { color: '#FFFFFF', fontSize: 20, fontWeight: '800' },

  // Footer
  footer: {
    padding: 20,
    backgroundColor: colors.surface,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  nextBtn: {
    backgroundColor: colors.success,
    borderRadius: 20,
    paddingVertical: 16,
    alignItems: 'center',
  },
  nextBtnText: { ...typography.body, fontWeight: '800', color: '#FFFFFF', fontSize: 17 },

  // Results
  resultsContainer: {
    flexGrow: 1,
    alignItems: 'center',
    padding: 32,
    backgroundColor: colors.background,
  },
  resultsEmoji: { fontSize: 72, marginBottom: 12 },
  resultsTitle: { ...typography.header, color: colors.text, marginBottom: 32, textAlign: 'center' },
  scoreCard: { marginBottom: 24 },
  scoreCircle: {
    width: 160,
    height: 160,
    borderRadius: 80,
    borderWidth: 8,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: colors.surface,
  },
  scoreNumber: { ...typography.header, fontSize: 32 },
  scorePercent: { ...typography.body, color: colors.textMuted },
  resultsMessage: { ...typography.subtitle, color: colors.text, textAlign: 'center', marginBottom: 32, lineHeight: 26 },

  reviewSection: { width: '100%', marginBottom: 32 },
  reviewTitle: { ...typography.subtitle, fontSize: 16, color: colors.text, marginBottom: 12 },
  reviewCard: {
    backgroundColor: colors.surface,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 14,
    marginBottom: 10,
  },
  reviewQuestion: { ...typography.body, fontWeight: '600', color: colors.text, marginBottom: 8, lineHeight: 21 },
  reviewAnswer: { ...typography.small, fontWeight: '700' },
  reviewCorrect: { ...typography.small, color: colors.success, fontWeight: '700', marginTop: 4 },
  reviewExplanation: { ...typography.small, color: colors.textMuted, marginTop: 8, lineHeight: 17 },

  doneBtn: {
    backgroundColor: colors.primary,
    borderRadius: 20,
    paddingHorizontal: 32,
    paddingVertical: 16,
  },
  doneBtnText: { ...typography.body, fontWeight: '800', color: '#FFFFFF', fontSize: 17 },
});
