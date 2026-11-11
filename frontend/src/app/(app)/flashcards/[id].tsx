import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, ScrollView, Alert, Animated } from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { flashcardService } from '../../../api/flashcards';
import { colors } from '../../../theme/colors';
import { typography } from '../../../theme/typography';

const GRADES: Array<{ key: 'AGAIN' | 'HARD' | 'GOOD' | 'EASY'; label: string; color: string }> = [
  { key: 'AGAIN', label: 'Again', color: colors.error },
  { key: 'HARD', label: 'Hard', color: '#D97706' },
  { key: 'GOOD', label: 'Good', color: colors.primary },
  { key: 'EASY', label: 'Easy', color: colors.success },
];

// A revision session across the whole exam's due cards — the Phase 10
// "smart revision" surface. `id` here is the examId, not a document.
export default function FlashcardReviewScreen() {
  const { id: examId } = useLocalSearchParams();
  const router = useRouter();

  const [cards, setCards] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [index, setIndex] = useState(0);
  const [flipped, setFlipped] = useState(false);
  const [grading, setGrading] = useState(false);
  const [reviewedCount, setReviewedCount] = useState(0);
  const [flipAnim] = useState(new Animated.Value(0));

  useEffect(() => {
    const fetchDue = async () => {
      try {
        const res: any = await flashcardService.getDueFlashcards(examId as string);
        setCards(res.data || []);
      } catch (error: any) {
        Alert.alert('Error', error.response?.data?.message || 'Failed to load cards due for review');
        router.back();
      } finally {
        setLoading(false);
      }
    };
    if (examId) fetchDue();
  }, [examId]);

  const handleFlip = () => {
    Animated.spring(flipAnim, { toValue: flipped ? 0 : 180, useNativeDriver: true, speed: 12, bounciness: 6 }).start();
    setFlipped(!flipped);
  };

  const handleGrade = async (grade: 'AGAIN' | 'HARD' | 'GOOD' | 'EASY') => {
    const card = cards[index];
    try {
      setGrading(true);
      await flashcardService.reviewFlashcard(card.id, grade);
      setReviewedCount((c) => c + 1);
      flipAnim.setValue(0);
      setFlipped(false);
      setIndex((i) => i + 1);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to record this review');
    } finally {
      setGrading(false);
    }
  };

  if (loading) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  if (cards.length === 0 || index >= cards.length) {
    const allCaughtUp = cards.length === 0;
    return (
      <SafeAreaView style={styles.safeArea}>
        <View style={styles.doneContainer}>
          <Text style={styles.doneEmoji}>{allCaughtUp ? '🎉' : '✅'}</Text>
          <Text style={styles.doneTitle}>{allCaughtUp ? "You're all caught up" : 'Session Complete'}</Text>
          <Text style={styles.doneSubtitle}>
            {allCaughtUp
              ? 'No flashcards are due for review right now — generate some from a topic, or check back later.'
              : `Reviewed ${reviewedCount} card${reviewedCount === 1 ? '' : 's'}.`}
          </Text>
          <TouchableOpacity style={styles.doneBtn} onPress={() => router.back()} activeOpacity={0.85}>
            <Text style={styles.doneBtnText}>Back →</Text>
          </TouchableOpacity>
        </View>
      </SafeAreaView>
    );
  }

  const card = cards[index];
  const progress = (index + 1) / cards.length;
  const frontRotate = flipAnim.interpolate({ inputRange: [0, 180], outputRange: ['0deg', '180deg'] });
  const backRotate = flipAnim.interpolate({ inputRange: [0, 180], outputRange: ['180deg', '360deg'] });

  return (
    <SafeAreaView style={styles.safeArea}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Revision</Text>
        <Text style={styles.headerProgress}>
          <Text style={styles.headerProgressCurrent}>{index + 1}</Text>/{cards.length}
        </Text>
      </View>

      <View style={styles.progressBarBg}>
        <View style={[styles.progressBarFill, { width: `${progress * 100}%` as any }]} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        <TouchableOpacity activeOpacity={0.9} onPress={handleFlip} style={styles.cardWrapper}>
          <Animated.View style={[styles.card, styles.cardFront, { transform: [{ rotateY: frontRotate }] }]}>
            <Text style={styles.cardLabel}>QUESTION</Text>
            <Text style={styles.cardText}>{card.front}</Text>
            {card.hint && <Text style={styles.cardHint}>💡 {card.hint}</Text>}
            <Text style={styles.tapHint}>Tap to reveal answer</Text>
          </Animated.View>
          <Animated.View style={[styles.card, styles.cardBack, { transform: [{ rotateY: backRotate }] }]}>
            <Text style={[styles.cardLabel, { color: colors.success }]}>ANSWER</Text>
            <Text style={styles.cardText}>{card.back}</Text>
          </Animated.View>
        </TouchableOpacity>
      </ScrollView>

      {flipped && (
        <View style={styles.footer}>
          <Text style={styles.gradePrompt}>How well did you know this?</Text>
          <View style={styles.gradeRow}>
            {GRADES.map((g) => (
              <TouchableOpacity
                key={g.key}
                style={[styles.gradeBtn, { backgroundColor: g.color }]}
                onPress={() => handleGrade(g.key)}
                disabled={grading}
                activeOpacity={0.85}
              >
                <Text style={styles.gradeBtnText}>{g.label}</Text>
              </TouchableOpacity>
            ))}
          </View>
        </View>
      )}
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
  headerProgress: { ...typography.body, color: colors.textMuted, fontWeight: '600', minWidth: 36, textAlign: 'right' },
  headerProgressCurrent: { color: colors.success, fontWeight: '800' },

  progressBarBg: {
    height: 6,
    backgroundColor: colors.border,
    marginHorizontal: 20,
    borderRadius: 3,
    marginBottom: 20,
    overflow: 'hidden',
  },
  progressBarFill: { height: '100%', backgroundColor: colors.success, borderRadius: 3 },

  scrollContent: { paddingHorizontal: 20, paddingBottom: 40, flexGrow: 1, justifyContent: 'center' },

  cardWrapper: { minHeight: 260 },
  card: {
    position: 'absolute',
    width: '100%',
    minHeight: 260,
    borderRadius: 24,
    padding: 28,
    justifyContent: 'center',
    alignItems: 'center',
    backfaceVisibility: 'hidden',
    borderWidth: 1,
    borderColor: colors.border,
  },
  cardFront: { backgroundColor: colors.surfaceWarm },
  cardBack: { backgroundColor: colors.successLight },
  cardLabel: { ...typography.label, color: colors.primary, marginBottom: 14 },
  cardText: { ...typography.title, fontSize: 20, color: colors.text, textAlign: 'center', lineHeight: 28 },
  cardHint: { ...typography.small, color: colors.textMuted, marginTop: 16, textAlign: 'center' },
  tapHint: { ...typography.small, color: colors.textLight, marginTop: 20, textAlign: 'center' },

  footer: { padding: 20, backgroundColor: colors.surface, borderTopWidth: 1, borderTopColor: colors.border },
  gradePrompt: { ...typography.small, color: colors.textMuted, textAlign: 'center', marginBottom: 10 },
  gradeRow: { flexDirection: 'row', gap: 8 },
  gradeBtn: { flex: 1, borderRadius: 14, paddingVertical: 14, alignItems: 'center' },
  gradeBtnText: { ...typography.caption, fontWeight: '800', color: '#FFFFFF' },

  doneContainer: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32 },
  doneEmoji: { fontSize: 64, marginBottom: 16 },
  doneTitle: { ...typography.header, color: colors.text, textAlign: 'center', marginBottom: 8 },
  doneSubtitle: { ...typography.body, color: colors.textMuted, textAlign: 'center', lineHeight: 22, marginBottom: 32 },
  doneBtn: { backgroundColor: colors.primary, borderRadius: 20, paddingHorizontal: 32, paddingVertical: 16 },
  doneBtnText: { ...typography.body, fontWeight: '800', color: '#FFFFFF', fontSize: 17 },
});
