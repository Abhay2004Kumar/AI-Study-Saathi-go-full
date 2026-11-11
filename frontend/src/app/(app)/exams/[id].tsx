import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, Alert } from 'react-native';
import { useLocalSearchParams, useRouter, useFocusEffect } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { examService } from '../../../api/exams';
import { documentService } from '../../../api/documents';
import { masteryService } from '../../../api/mastery';
import { flashcardService } from '../../../api/flashcards';
import AppTextInput from '../../../components/AppTextInput';
import AppButton from '../../../components/AppButton';
import AppCard from '../../../components/AppCard';
import SyllabusTree from '../../../components/SyllabusTree';
import KeyboardAwareScrollView from '../../../components/KeyboardAwareScrollView';
import { type SubjectNode, fromApiSubjects, toApiSubjects } from '../../../utils/syllabusTree';
import { colors } from '../../../theme/colors';
import { typography } from '../../../theme/typography';

export default function ExamDetailScreen() {
  const { id } = useLocalSearchParams();
  const examId = typeof id === 'string' ? id : '';
  const router = useRouter();

  const [exam, setExam] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const [editingDetails, setEditingDetails] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [examDate, setExamDate] = useState('');
  const [hoursPerDay, setHoursPerDay] = useState('');
  const [savingDetails, setSavingDetails] = useState(false);

  const [editingSyllabus, setEditingSyllabus] = useState(false);
  const [subjects, setSubjects] = useState<SubjectNode[]>([]);
  const [savingSyllabus, setSavingSyllabus] = useState(false);

  const [resources, setResources] = useState<any[]>([]);
  const [resourcesLoading, setResourcesLoading] = useState(true);

  const [weakTopics, setWeakTopics] = useState<any[]>([]);
  const [dueCardCount, setDueCardCount] = useState(0);

  const loadExam = async () => {
    try {
      const res: any = await examService.getExam(examId);
      setExam(res.data);
      setName(res.data.name || '');
      setDescription(res.data.description || '');
      setExamDate(res.data.examDate ? res.data.examDate.slice(0, 10) : '');
      setHoursPerDay(res.data.availableHoursPerDay != null ? String(res.data.availableHoursPerDay) : '');
      setSubjects(fromApiSubjects(res.data.subjects || []));
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to load this exam');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const loadResources = async () => {
    try {
      const res: any = await documentService.getDocuments({ examId });
      setResources(res.data || []);
    } catch (error) {
      console.log('Failed to load exam resources', error);
    } finally {
      setResourcesLoading(false);
    }
  };

  const loadWeakTopics = async () => {
    try {
      const res: any = await masteryService.getExamMastery(examId);
      // Only surface topics the student has actually attempted — an
      // un-attempted topic isn't "weak", just not started yet.
      setWeakTopics((res.data || []).filter((t: any) => t.mastery.attemptCount > 0 && t.mastery.level !== 'STRONG').slice(0, 3));
    } catch (error) {
      console.log('Failed to load weak topics', error);
    }
  };

  const loadDueCardCount = async () => {
    try {
      const res: any = await flashcardService.getDueFlashcards(examId, 50);
      setDueCardCount((res.data || []).length);
    } catch (error) {
      console.log('Failed to load due flashcard count', error);
    }
  };

  useFocusEffect(
    useCallback(() => {
      if (examId) {
        loadExam();
        loadResources();
        loadWeakTopics();
        loadDueCardCount();
      }
    }, [examId])
  );

  const handleDeleteResource = (docId: string, title: string) => {
    Alert.alert('Delete Document', `Are you sure you want to delete "${title}"?`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: async () => {
          try {
            await documentService.deleteDocument(docId);
            loadResources();
          } catch (error: any) {
            Alert.alert('Error', error.response?.data?.message || 'Failed to delete document');
          }
        },
      },
    ]);
  };

  const handleRetryResource = async (docId: string) => {
    try {
      await documentService.retryDocument(docId);
      loadResources();
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to retry processing');
    }
  };

  const handleSaveDetails = async () => {
    if (!name.trim()) {
      Alert.alert('Missing Name', 'Exam name cannot be empty');
      return;
    }
    const parsedHours = hoursPerDay ? Number(hoursPerDay) : null;
    if (hoursPerDay && (Number.isNaN(parsedHours) || parsedHours! <= 0)) {
      Alert.alert('Invalid Hours', 'Available hours per day must be a positive number');
      return;
    }
    if (examDate && Number.isNaN(Date.parse(examDate))) {
      Alert.alert('Invalid Date', 'Exam date should look like YYYY-MM-DD');
      return;
    }

    try {
      setSavingDetails(true);
      const res: any = await examService.updateExam(examId, {
        name: name.trim(),
        description: description.trim() || null,
        examDate: examDate.trim() || null,
        availableHoursPerDay: parsedHours,
      });
      setExam(res.data);
      setEditingDetails(false);
    } catch (error: any) {
      Alert.alert('Could Not Save', error.response?.data?.message || error.message);
    } finally {
      setSavingDetails(false);
    }
  };

  const handleSaveSyllabus = async () => {
    try {
      setSavingSyllabus(true);
      const res: any = await examService.replaceSyllabus(examId, toApiSubjects(subjects));
      setExam(res.data);
      setSubjects(fromApiSubjects(res.data.subjects || []));
      setEditingSyllabus(false);
    } catch (error: any) {
      Alert.alert('Could Not Save Syllabus', error.response?.data?.message || error.message);
    } finally {
      setSavingSyllabus(false);
    }
  };

  const handleCancelSyllabusEdit = () => {
    setSubjects(fromApiSubjects(exam?.subjects || []));
    setEditingSyllabus(false);
  };

  const handleDelete = () => {
    Alert.alert('Delete Exam', `Are you sure you want to delete "${exam?.name}"? This removes its whole syllabus too.`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: async () => {
          try {
            await examService.deleteExam(examId);
            router.replace('/(app)/exams' as any);
          } catch (error: any) {
            Alert.alert('Error', error.response?.data?.message || 'Failed to delete exam');
          }
        },
      },
    ]);
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return null;
    return new Date(dateStr).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' });
  };

  if (loading || !exam) {
    return (
      <SafeAreaView style={styles.loadingContainer} edges={['top', 'bottom']}>
        <ActivityIndicator size="large" color={colors.primary} />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle} numberOfLines={1}>{exam.name}</Text>
        <TouchableOpacity onPress={handleDelete} style={styles.headerDelete} hitSlop={8}>
          <Text style={styles.headerDeleteText}>🗑</Text>
        </TouchableOpacity>
      </View>

      <KeyboardAwareScrollView contentContainerStyle={styles.scrollContent}>
        {editingDetails ? (
          <View style={styles.detailsCard}>
            <AppTextInput label="Exam Name" value={name} onChangeText={setName} icon="🎯" />
            <AppTextInput label="Description" value={description} onChangeText={setDescription} icon="📝" />
            <View style={styles.row}>
              <View style={styles.rowItem}>
                <AppTextInput label="Exam Date" placeholder="YYYY-MM-DD" value={examDate} onChangeText={setExamDate} icon="📅" />
              </View>
              <View style={styles.rowItem}>
                <AppTextInput label="Hours/Day" value={hoursPerDay} onChangeText={setHoursPerDay} keyboardType="numeric" icon="⏱" />
              </View>
            </View>
            <View style={styles.editActions}>
              <AppButton title="Cancel" variant="ghost" onPress={() => setEditingDetails(false)} style={styles.editActionBtn} />
              <AppButton title={savingDetails ? 'Saving...' : 'Save'} loading={savingDetails} onPress={handleSaveDetails} style={styles.editActionBtn} />
            </View>
          </View>
        ) : (
          <View style={styles.detailsCard}>
            {exam.description ? <Text style={styles.description}>{exam.description}</Text> : null}
            <View style={styles.metaRow}>
              {formatDate(exam.examDate) && (
                <View style={styles.metaPill}>
                  <Text style={styles.metaPillText}>📅 {formatDate(exam.examDate)}</Text>
                </View>
              )}
              {exam.availableHoursPerDay != null && (
                <View style={styles.metaPill}>
                  <Text style={styles.metaPillText}>⏱ {exam.availableHoursPerDay} hrs/day</Text>
                </View>
              )}
            </View>
            <TouchableOpacity onPress={() => setEditingDetails(true)} style={styles.editDetailsLink}>
              <Text style={styles.editDetailsLinkText}>Edit Details</Text>
            </TouchableOpacity>
          </View>
        )}

        <TouchableOpacity
          style={styles.askBanner}
          onPress={() => router.push({ pathname: '/(app)/exams/[id]/assistant', params: { id: examId } } as any)}
          activeOpacity={0.85}
        >
          <Text style={styles.askBannerEmoji}>🧑‍🏫</Text>
          <Text style={styles.askBannerText}>Study Assistant</Text>
          <Text style={styles.chevronText}>›</Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.planBanner}
          onPress={() => router.push({ pathname: '/(app)/ask', params: { examId, examName: exam.name } } as any)}
          activeOpacity={0.85}
        >
          <Text style={styles.askBannerEmoji}>📄</Text>
          <Text style={styles.askBannerText}>Quick Ask (your materials only)</Text>
          <Text style={styles.chevronText}>›</Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.planBanner}
          onPress={() => router.push({ pathname: '/(app)/exams/[id]/plan', params: { id: examId } } as any)}
          activeOpacity={0.85}
        >
          <Text style={styles.askBannerEmoji}>📅</Text>
          <Text style={styles.askBannerText}>Study Plan</Text>
          <Text style={styles.chevronText}>›</Text>
        </TouchableOpacity>

        {dueCardCount > 0 && (
          <TouchableOpacity
            style={styles.reviseBanner}
            onPress={() => router.push({ pathname: '/(app)/flashcards/[id]', params: { id: examId } } as any)}
            activeOpacity={0.85}
          >
            <Text style={styles.askBannerEmoji}>🔁</Text>
            <Text style={styles.askBannerText}>
              {dueCardCount} flashcard{dueCardCount === 1 ? '' : 's'} due for revision
            </Text>
            <Text style={styles.chevronText}>›</Text>
          </TouchableOpacity>
        )}

        {weakTopics.length > 0 && (
          <View style={styles.weakCard}>
            <Text style={styles.weakCardTitle}>🎯 Focus Here</Text>
            <Text style={styles.weakCardSubtitle}>Topics your quiz attempts show room to grow</Text>
            {weakTopics.map((t) => (
              <TouchableOpacity
                key={t.topic.id}
                style={styles.weakTopicRow}
                onPress={() =>
                  router.push({ pathname: '/(app)/exams/[id]/topics/[topicId]', params: { id: examId, topicId: t.topic.id } } as any)
                }
                activeOpacity={0.8}
              >
                <Text style={styles.weakTopicName} numberOfLines={1}>{t.topic.name}</Text>
                <Text style={[styles.weakTopicLevel, { color: t.mastery.level === 'WEAK' ? colors.error : '#D97706' }]}>
                  {t.mastery.level === 'WEAK' ? 'Weak' : 'Developing'}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        )}

        <View style={styles.sectionRow}>
          <Text style={styles.sectionTitle}>Syllabus</Text>
          {!editingSyllabus && (
            <TouchableOpacity onPress={() => setEditingSyllabus(true)}>
              <Text style={styles.editDetailsLinkText}>Edit Syllabus</Text>
            </TouchableOpacity>
          )}
        </View>

        <SyllabusTree
          subjects={subjects}
          onChange={editingSyllabus ? setSubjects : undefined}
          readOnly={!editingSyllabus}
          onTopicPress={
            editingSyllabus
              ? undefined
              : (topic) => {
                  // Only saved topics (with a real server id) have anything to browse.
                  if (!topic.id) return;
                  router.push({
                    pathname: '/(app)/exams/[id]/topics/[topicId]',
                    params: { id: examId, topicId: topic.id },
                  } as any);
                }
          }
        />

        {editingSyllabus && (
          <View style={styles.editActions}>
            <AppButton title="Cancel" variant="ghost" onPress={handleCancelSyllabusEdit} style={styles.editActionBtn} />
            <AppButton
              title={savingSyllabus ? 'Saving...' : 'Save Syllabus'}
              loading={savingSyllabus}
              onPress={handleSaveSyllabus}
              style={styles.editActionBtn}
            />
          </View>
        )}

        <View style={styles.sectionRow}>
          <Text style={styles.sectionTitle}>Resources</Text>
          <TouchableOpacity onPress={() => router.push({ pathname: '/(app)/upload', params: { examId } } as any)}>
            <Text style={styles.editDetailsLinkText}>+ Upload</Text>
          </TouchableOpacity>
        </View>

        {resourcesLoading ? (
          <ActivityIndicator color={colors.primary} style={{ marginBottom: 20 }} />
        ) : resources.length === 0 ? (
          <Text style={styles.emptyResourcesText}>No resources linked to this exam yet.</Text>
        ) : (
          resources.map((doc) => (
            <AppCard
              key={doc.id}
              title={doc.title}
              subtitle={doc.fileName}
              status={doc.processingStatus}
              category={doc.category}
              date={doc.createdAt}
              onPress={() => router.push({ pathname: '/(app)/documents/[id]', params: { id: doc.id } } as any)}
              actions={
                doc.processingStatus === 'READY'
                  ? [
                      {
                        label: '🤖 Tutor',
                        onPress: () =>
                          router.push({ pathname: '/(app)/tutoring/[id]', params: { id: doc.id, title: doc.title } } as any),
                      },
                      { label: '🗑', variant: 'danger', compact: true, onPress: () => handleDeleteResource(doc.id, doc.title) },
                    ]
                  : doc.processingStatus === 'FAILED'
                    ? [
                        { label: '🔄 Retry', onPress: () => handleRetryResource(doc.id) },
                        { label: '🗑 Delete', variant: 'danger', onPress: () => handleDeleteResource(doc.id, doc.title) },
                      ]
                    : [{ label: '🗑 Delete', variant: 'danger', onPress: () => handleDeleteResource(doc.id, doc.title) }]
              }
            />
          ))
        )}
      </KeyboardAwareScrollView>
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
  headerDelete: {
    width: 36, height: 36, borderRadius: 18,
    backgroundColor: colors.surfaceWarm,
    justifyContent: 'center', alignItems: 'center',
  },
  headerDeleteText: { fontSize: 16 },

  scrollContent: {
    paddingHorizontal: 20,
    paddingBottom: 40,
    width: '100%',
    maxWidth: 520,
    alignSelf: 'center',
  },

  detailsCard: {
    backgroundColor: colors.surface,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 18,
    marginBottom: 20,
  },
  description: { ...typography.body, color: colors.textMuted, marginBottom: 12, lineHeight: 22 },
  metaRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  metaPill: {
    backgroundColor: colors.surfaceWarm,
    borderRadius: 20,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  metaPillText: { ...typography.small, color: colors.text, fontWeight: '600' },
  editDetailsLink: { marginTop: 12 },
  editDetailsLinkText: { ...typography.caption, color: colors.primary, fontWeight: '700' },

  askBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.successLight,
    borderRadius: 16,
    paddingHorizontal: 16,
    paddingVertical: 14,
    marginBottom: 20,
    gap: 10,
  },
  askBannerEmoji: { fontSize: 18 },
  askBannerText: { ...typography.body, fontWeight: '700', color: colors.text, flex: 1 },
  chevronText: { fontSize: 20, color: colors.textMuted },
  planBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surfaceWarm,
    borderRadius: 16,
    paddingHorizontal: 16,
    paddingVertical: 14,
    marginBottom: 20,
    gap: 10,
    borderWidth: 1,
    borderColor: colors.border,
  },
  reviseBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#FEF3C7',
    borderRadius: 16,
    paddingHorizontal: 16,
    paddingVertical: 14,
    marginBottom: 20,
    gap: 10,
  },

  weakCard: {
    backgroundColor: colors.surface,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 16,
    marginBottom: 20,
  },
  weakCardTitle: { ...typography.subtitle, fontSize: 15, color: colors.text },
  weakCardSubtitle: { ...typography.small, color: colors.textMuted, marginTop: 4, marginBottom: 10 },
  weakTopicRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 10,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  weakTopicName: { ...typography.body, color: colors.text, fontWeight: '600', flex: 1, marginRight: 8 },
  weakTopicLevel: { ...typography.small, fontWeight: '800' },

  row: { flexDirection: 'row', gap: 12 },
  rowItem: { flex: 1 },

  sectionRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  sectionTitle: { ...typography.subtitle, color: colors.text },

  editActions: { flexDirection: 'row', gap: 12, marginTop: 8, marginBottom: 20 },
  editActionBtn: { flex: 1, marginVertical: 0 },

  emptyResourcesText: { ...typography.body, color: colors.textMuted, marginBottom: 20 },
});
