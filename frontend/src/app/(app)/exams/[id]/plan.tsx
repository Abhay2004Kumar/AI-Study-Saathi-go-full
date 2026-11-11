import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, ScrollView, Alert } from 'react-native';
import { useLocalSearchParams, useRouter, useFocusEffect } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { studyPlanService } from '../../../../api/studyPlan';
import AppButton from '../../../../components/AppButton';
import { colors } from '../../../../theme/colors';
import { typography } from '../../../../theme/typography';

const formatDayHeader = (dateStr: string) => {
  const date = new Date(dateStr);
  const today = new Date();
  if (date.toDateString() === today.toDateString()) return 'Today';
  const tomorrow = new Date(today);
  tomorrow.setDate(tomorrow.getDate() + 1);
  if (date.toDateString() === tomorrow.toDateString()) return 'Tomorrow';
  return date.toLocaleDateString('en-IN', { weekday: 'short', day: 'numeric', month: 'short' });
};

const groupByDate = (sessions: any[]) => {
  const groups: { date: string; sessions: any[] }[] = [];
  sessions.forEach((s) => {
    const key = new Date(s.date).toDateString();
    let group = groups.find((g) => new Date(g.date).toDateString() === key);
    if (!group) {
      group = { date: s.date, sessions: [] };
      groups.push(group);
    }
    group.sessions.push(s);
  });
  return groups;
};

export default function StudyPlanScreen() {
  const { id: examId } = useLocalSearchParams();
  const router = useRouter();

  const [plan, setPlan] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const load = async () => {
    try {
      const res: any = await studyPlanService.getPlan(examId as string);
      setPlan(res.data);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to load the study plan');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  useFocusEffect(
    useCallback(() => {
      if (examId) load();
    }, [examId])
  );

  const handleGenerate = async () => {
    try {
      setGenerating(true);
      const res: any = await studyPlanService.generatePlan(examId as string);
      setPlan(res.data.plan);
      if (res.data.skippedTopicIds?.length > 0) {
        Alert.alert(
          'Plan Ready',
          `${res.data.skippedTopicIds.length} topic(s) you've already mastered were skipped — this plan focuses on what still needs work.`
        );
      }
    } catch (error: any) {
      Alert.alert('Could Not Generate Plan', error.response?.data?.message || 'Something went wrong');
    } finally {
      setGenerating(false);
    }
  };

  const handleToggle = async (session: any) => {
    try {
      setTogglingId(session.id);
      const res: any = await studyPlanService.setSessionCompletion(session.id, !session.completed);
      setPlan((prev: any) => ({
        ...prev,
        sessions: prev.sessions.map((s: any) => (s.id === session.id ? res.data : s)),
      }));
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to update this session');
    } finally {
      setTogglingId(null);
    }
  };

  if (loading) {
    return (
      <SafeAreaView style={styles.loadingContainer} edges={['top', 'bottom']}>
        <ActivityIndicator size="large" color={colors.primary} />
      </SafeAreaView>
    );
  }

  const dayGroups = plan ? groupByDate(plan.sessions) : [];

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Study Plan</Text>
        <View style={{ width: 36 }} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        {!plan ? (
          <View style={styles.emptyState}>
            <Text style={styles.emptyEmoji}>📅</Text>
            <Text style={styles.emptyTitle}>No study plan yet</Text>
            <Text style={styles.emptyText}>
              Generate a day-by-day plan built from your weak topics and how often they&apos;ve shown up in past papers.
              Make sure this exam has a date and your available hours/day set in Edit Details first.
            </Text>
            <AppButton title={generating ? 'Generating...' : 'Generate Study Plan'} loading={generating} onPress={handleGenerate} />
          </View>
        ) : (
          <>
            <View style={styles.regenerateRow}>
              <Text style={styles.rangeText}>
                {new Date(plan.startDate).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })} –{' '}
                {new Date(plan.endDate).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })}
              </Text>
              <TouchableOpacity onPress={handleGenerate} disabled={generating}>
                <Text style={styles.regenerateLink}>{generating ? 'Regenerating...' : '↻ Regenerate'}</Text>
              </TouchableOpacity>
            </View>

            {dayGroups.length === 0 ? (
              <View style={styles.emptyState}>
                <Text style={styles.emptyEmoji}>🎉</Text>
                <Text style={styles.emptyTitle}>Nothing scheduled</Text>
                <Text style={styles.emptyText}>Every topic here already looks confidently mastered. Nice work.</Text>
              </View>
            ) : (
              dayGroups.map((group) => {
                const totalMinutes = group.sessions.reduce((s: number, x: any) => s + x.durationMinutes, 0);
                return (
                  <View key={group.date} style={styles.dayGroup}>
                    <View style={styles.dayHeaderRow}>
                      <Text style={styles.dayHeader}>{formatDayHeader(group.date)}</Text>
                      <Text style={styles.dayTotal}>{Math.round((totalMinutes / 60) * 10) / 10}h</Text>
                    </View>
                    {group.sessions.map((s: any) => (
                      <View key={s.id} style={styles.sessionCard}>
                        <TouchableOpacity
                          style={styles.checkbox}
                          onPress={() => handleToggle(s)}
                          disabled={togglingId === s.id}
                        >
                          <View style={[styles.checkboxBox, s.completed && styles.checkboxBoxChecked]}>
                            {s.completed && <Text style={styles.checkboxMark}>✓</Text>}
                          </View>
                        </TouchableOpacity>
                        <TouchableOpacity
                          style={{ flex: 1 }}
                          onPress={() =>
                            router.push({ pathname: '/(app)/exams/[id]/topics/[topicId]', params: { id: examId, topicId: s.topicId } } as any)
                          }
                        >
                          <Text style={[styles.sessionTopic, s.completed && styles.sessionTopicDone]} numberOfLines={1}>
                            {s.topicName || s.topic?.name}
                          </Text>
                          <Text style={styles.sessionMeta}>{s.durationMinutes} min · {s.reason}</Text>
                        </TouchableOpacity>
                      </View>
                    ))}
                  </View>
                );
              })
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

  emptyState: { alignItems: 'center', paddingTop: 40, paddingHorizontal: 12, gap: 8 },
  emptyEmoji: { fontSize: 48, marginBottom: 8 },
  emptyTitle: { ...typography.subtitle, color: colors.text },
  emptyText: { ...typography.body, color: colors.textMuted, textAlign: 'center', lineHeight: 22, marginBottom: 16 },

  regenerateRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 20,
  },
  rangeText: { ...typography.small, color: colors.textMuted, fontWeight: '600' },
  regenerateLink: { ...typography.caption, color: colors.primary, fontWeight: '700' },

  dayGroup: { marginBottom: 20 },
  dayHeaderRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 8 },
  dayHeader: { ...typography.subtitle, fontSize: 15, color: colors.text },
  dayTotal: { ...typography.small, color: colors.textMuted, fontWeight: '700' },

  sessionCard: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    backgroundColor: colors.surface,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 12,
    marginBottom: 8,
    gap: 10,
  },
  checkbox: { paddingTop: 2 },
  checkboxBox: {
    width: 22, height: 22, borderRadius: 11,
    borderWidth: 2, borderColor: colors.border,
    justifyContent: 'center', alignItems: 'center',
  },
  checkboxBoxChecked: { backgroundColor: colors.success, borderColor: colors.success },
  checkboxMark: { color: '#FFFFFF', fontSize: 13, fontWeight: '800' },
  sessionTopic: { ...typography.body, fontWeight: '600', color: colors.text },
  sessionTopicDone: { textDecorationLine: 'line-through', color: colors.textMuted },
  sessionMeta: { ...typography.small, color: colors.textMuted, marginTop: 2 },
});
