import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, Alert, ScrollView } from 'react-native';
import { useLocalSearchParams, useRouter, useFocusEffect } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { documentService } from '../../../api/documents';
import { mappingService } from '../../../api/mappings';
import { pyqService } from '../../../api/pyq';
import SyllabusTree from '../../../components/SyllabusTree';
import { fromApiSubjects } from '../../../utils/syllabusTree';
import { colors } from '../../../theme/colors';
import { typography } from '../../../theme/typography';

const CATEGORY_LABELS: Record<string, string> = {
  SYLLABUS: '📘 Syllabus',
  NOTES: '📝 Notes',
  PYQ: '🗞️ Past Papers',
  GENERAL: '📄 General',
};

const STATUS_CONFIG: Record<string, { label: string; bg: string; color: string }> = {
  READY: { label: 'READY', bg: colors.successLight, color: colors.success },
  PROCESSING: { label: 'PROCESSING', bg: '#FEF3C7', color: '#D97706' },
  UPLOADED: { label: 'UPLOADED', bg: colors.surfaceWarm, color: colors.textMuted },
  FAILED: { label: 'FAILED', bg: '#FEE2E2', color: colors.error },
};

export default function DocumentDetailScreen() {
  const { id } = useLocalSearchParams();
  const documentId = typeof id === 'string' ? id : '';
  const router = useRouter();

  const [doc, setDoc] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [applyingCategory, setApplyingCategory] = useState(false);

  const [mappings, setMappings] = useState<any[]>([]);
  const [mappingsLoading, setMappingsLoading] = useState(false);
  const [mapping, setMapping] = useState(false);

  const [pyqQuestions, setPyqQuestions] = useState<any[]>([]);
  const [analyzingPyq, setAnalyzingPyq] = useState(false);

  const loadDocument = async () => {
    try {
      const res: any = await documentService.getDocument(documentId);
      setDoc(res.data);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to load this document');
      router.back();
    } finally {
      setLoading(false);
    }
  };

  const loadMappings = async () => {
    try {
      setMappingsLoading(true);
      const res: any = await mappingService.getDocumentMappings(documentId);
      setMappings(res.data || []);
    } catch (error) {
      console.log('Failed to load topic mappings', error);
    } finally {
      setMappingsLoading(false);
    }
  };

  const loadPyqQuestions = async () => {
    try {
      const res: any = await pyqService.getDocumentQuestions(documentId);
      setPyqQuestions(res.data || []);
    } catch (error) {
      console.log('Failed to load PYQ questions', error);
    }
  };

  useFocusEffect(
    useCallback(() => {
      if (documentId) {
        loadDocument();
        loadMappings();
        loadPyqQuestions();
      }
    }, [documentId])
  );

  const handleMapTopics = async () => {
    try {
      setMapping(true);
      await mappingService.mapDocumentTopics(documentId);
      await loadMappings();
    } catch (error: any) {
      Alert.alert('Could Not Map Topics', error.response?.data?.message || error.message);
    } finally {
      setMapping(false);
    }
  };

  const handleAnalyzePyq = async () => {
    try {
      setAnalyzingPyq(true);
      await pyqService.analyzePyqDocument(documentId);
      await loadPyqQuestions();
    } catch (error: any) {
      Alert.alert('Could Not Analyze Questions', error.response?.data?.message || error.message);
    } finally {
      setAnalyzingPyq(false);
    }
  };

  const handleAcceptAiCategory = async () => {
    if (!doc?.aiCategory) return;
    try {
      setApplyingCategory(true);
      const res: any = await documentService.updateDocumentCategory(documentId, doc.aiCategory);
      setDoc((prev: any) => ({ ...prev, ...res.data }));
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to update category');
    } finally {
      setApplyingCategory(false);
    }
  };

  const formatSize = (bytes?: number) => {
    if (!bytes) return '—';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  if (loading || !doc) {
    return (
      <SafeAreaView style={styles.loadingContainer} edges={['top', 'bottom']}>
        <ActivityIndicator size="large" color={colors.primary} />
      </SafeAreaView>
    );
  }

  const statusConfig = STATUS_CONFIG[doc.processingStatus] || STATUS_CONFIG.UPLOADED;
  const extraction = doc.extraction;
  // Only surface the AI's opinion when it disagrees with the user's choice
  // and it's actually confident — a low-confidence guess isn't worth
  // interrupting the user over.
  const showAiSuggestion =
    doc.aiCategory && doc.aiCategory !== doc.category && (doc.classificationConfidence ?? 0) >= 0.5;

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle} numberOfLines={1}>{doc.title}</Text>
        <View style={{ width: 36 }} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        <View style={styles.metaCard}>
          <View style={styles.metaRow}>
            <View style={[styles.badge, { backgroundColor: statusConfig.bg }]}>
              <Text style={[styles.badgeText, { color: statusConfig.color }]}>{statusConfig.label}</Text>
            </View>
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{CATEGORY_LABELS[doc.category] || doc.category}</Text>
            </View>
          </View>
          <Text style={styles.fileInfo}>{doc.fileName} · {formatSize(doc.fileSize)}{doc.pageCount ? ` · ${doc.pageCount} pages` : ''}</Text>

          {showAiSuggestion && (
            <View style={styles.aiSuggestionCard}>
              <Text style={styles.aiSuggestionText}>
                ✨ AI thinks this looks more like <Text style={styles.aiSuggestionBold}>{CATEGORY_LABELS[doc.aiCategory] || doc.aiCategory}</Text>
                {' '}({Math.round((doc.classificationConfidence || 0) * 100)}% confident)
              </Text>
              <TouchableOpacity
                onPress={handleAcceptAiCategory}
                disabled={applyingCategory}
                style={styles.aiSuggestionBtn}
              >
                <Text style={styles.aiSuggestionBtnText}>{applyingCategory ? 'Applying…' : 'Use This Category'}</Text>
              </TouchableOpacity>
            </View>
          )}
        </View>

        {doc.examId && doc.processingStatus === 'READY' && (
          <View style={styles.section}>
            <View style={styles.mappingHeaderRow}>
              <Text style={styles.sectionTitle}>Syllabus Mapping</Text>
              <TouchableOpacity onPress={handleMapTopics} disabled={mapping}>
                <Text style={styles.mapLinkText}>
                  {mapping ? 'Mapping…' : mappings.length > 0 ? 'Re-map' : 'Map to Syllabus'}
                </Text>
              </TouchableOpacity>
            </View>

            {mappingsLoading ? (
              <ActivityIndicator color={colors.primary} />
            ) : mappings.length === 0 ? (
              <Text style={styles.mappingEmptyText}>
                Not mapped to any syllabus topic yet. Tap &quot;Map to Syllabus&quot; to have AI find which topics this covers.
              </Text>
            ) : (
              mappings.map((m) => (
                <TouchableOpacity
                  key={m.id}
                  style={[styles.mappingRow, m.status === 'PENDING_REVIEW' && styles.mappingRowPending]}
                  onPress={() =>
                    router.push({
                      pathname: '/(app)/exams/[id]/topics/[topicId]',
                      params: { id: doc.examId, topicId: m.topicId },
                    } as any)
                  }
                  activeOpacity={0.75}
                >
                  <Text style={styles.mappingTopicName} numberOfLines={1}>
                    {m.subtopic?.name || m.topic?.name || 'Unknown topic'}
                  </Text>
                  <Text style={styles.mappingPages}>
                    {m.startPage ? (m.endPage && m.endPage !== m.startPage ? `p.${m.startPage}–${m.endPage}` : `p.${m.startPage}`) : ''}
                  </Text>
                  {m.status === 'PENDING_REVIEW' && <Text style={styles.mappingPendingTag}>REVIEW</Text>}
                </TouchableOpacity>
              ))
            )}
          </View>
        )}

        {doc.processingStatus === 'PROCESSING' || doc.processingStatus === 'UPLOADED' ? (
          <View style={styles.pendingCard}>
            <ActivityIndicator color={colors.primary} />
            <Text style={styles.pendingText}>Still processing — insights will appear here once it&apos;s ready.</Text>
          </View>
        ) : !extraction ? (
          <View style={styles.pendingCard}>
            <Text style={styles.pendingText}>No AI insights available for this document yet.</Text>
          </View>
        ) : (
          <>
            {extraction.summary ? (
              <View style={styles.summaryCard}>
                <Text style={styles.sectionTitle}>Summary</Text>
                <Text style={styles.summaryText}>{extraction.summary}</Text>
              </View>
            ) : null}

            {extraction.resourceType === 'NOTES' && Array.isArray(extraction.topics) && extraction.topics.length > 0 && (
              <View style={styles.section}>
                <Text style={styles.sectionTitle}>Topics Covered</Text>
                {extraction.topics.map((t: any, i: number) => (
                  <View key={i} style={styles.topicRow}>
                    <Text style={styles.topicBullet}>•</Text>
                    <View style={{ flex: 1 }}>
                      <Text style={styles.topicName}>{t.name}</Text>
                      {t.description ? <Text style={styles.topicDescription}>{t.description}</Text> : null}
                    </View>
                  </View>
                ))}
              </View>
            )}

            {extraction.resourceType === 'SYLLABUS' && extraction.topics?.subjects?.length > 0 && (
              <View style={styles.section}>
                <Text style={styles.sectionTitle}>Extracted Structure</Text>
                <SyllabusTree subjects={fromApiSubjects(extraction.topics.subjects)} readOnly />
              </View>
            )}

            {extraction.resourceType === 'PYQ' && Array.isArray(extraction.questions) && extraction.questions.length > 0 && (
              <View style={styles.section}>
                <View style={styles.mappingHeaderRow}>
                  <Text style={styles.sectionTitle}>
                    {pyqQuestions.length > 0 ? `Questions by Topic (${pyqQuestions.length})` : `Extracted Questions (${extraction.questions.length})`}
                  </Text>
                  {doc.examId && (
                    <TouchableOpacity onPress={handleAnalyzePyq} disabled={analyzingPyq}>
                      <Text style={styles.mapLinkText}>
                        {analyzingPyq ? 'Analyzing…' : pyqQuestions.length > 0 ? 'Re-analyze' : 'Analyze by Topic'}
                      </Text>
                    </TouchableOpacity>
                  )}
                </View>

                {pyqQuestions.length > 0
                  ? pyqQuestions.map((q: any, i: number) => (
                      <TouchableOpacity
                        key={q.id}
                        style={[styles.questionCard, q.status === 'PENDING_REVIEW' && styles.questionCardPending]}
                        disabled={!q.topicId}
                        onPress={() =>
                          q.topicId &&
                          router.push({
                            pathname: '/(app)/exams/[id]/topics/[topicId]',
                            params: { id: doc.examId, topicId: q.topicId },
                          } as any)
                        }
                        activeOpacity={0.75}
                      >
                        <Text style={styles.questionText}>{i + 1}. {q.questionText}</Text>
                        {q.correctAnswer ? <Text style={styles.answerText}>✓ {q.correctAnswer}</Text> : null}
                        <View style={styles.questionMetaRow}>
                          {q.year ? <Text style={styles.yearText}>{q.year}</Text> : null}
                          {q.status === 'PENDING_REVIEW' && <Text style={styles.mappingPendingTag}>NEEDS REVIEW</Text>}
                          {!q.topicId && <Text style={styles.yearText}>No topic match</Text>}
                        </View>
                      </TouchableOpacity>
                    ))
                  : extraction.questions.map((q: any, i: number) => (
                      <View key={i} style={styles.questionCard}>
                        <Text style={styles.questionText}>{i + 1}. {q.question}</Text>
                        {Array.isArray(q.options) && q.options.length > 0 && (
                          <View style={styles.optionsList}>
                            {q.options.map((opt: string, oi: number) => (
                              <Text key={oi} style={styles.optionText}>{opt}</Text>
                            ))}
                          </View>
                        )}
                        {q.correctAnswer ? <Text style={styles.answerText}>✓ {q.correctAnswer}</Text> : null}
                        {q.year ? <Text style={styles.yearText}>{q.year}</Text> : null}
                      </View>
                    ))}
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

  metaCard: {
    backgroundColor: colors.surface,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 18,
    marginBottom: 20,
  },
  metaRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginBottom: 10 },
  badge: {
    backgroundColor: colors.surfaceWarm,
    borderRadius: 20,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  badgeText: { ...typography.small, fontWeight: '700', color: colors.text },
  fileInfo: { ...typography.small, color: colors.textMuted },

  aiSuggestionCard: {
    marginTop: 14,
    backgroundColor: colors.primaryLight,
    borderRadius: 14,
    padding: 14,
  },
  aiSuggestionText: { ...typography.caption, color: colors.primaryDark, lineHeight: 20 },
  aiSuggestionBold: { fontWeight: '800' },
  aiSuggestionBtn: {
    marginTop: 10,
    backgroundColor: colors.primary,
    borderRadius: 12,
    paddingVertical: 10,
    alignItems: 'center',
  },
  aiSuggestionBtnText: { ...typography.caption, fontWeight: '700', color: '#FFFFFF' },

  pendingCard: {
    backgroundColor: colors.surface,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 24,
    alignItems: 'center',
    gap: 12,
  },
  pendingText: { ...typography.body, color: colors.textMuted, textAlign: 'center' },

  summaryCard: {
    backgroundColor: colors.surfaceWarm,
    borderRadius: 16,
    padding: 16,
    marginBottom: 20,
  },
  summaryText: { ...typography.body, color: colors.text, lineHeight: 22, marginTop: 6 },

  section: { marginBottom: 20 },
  sectionTitle: { ...typography.subtitle, fontSize: 16, color: colors.text },

  mappingHeaderRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 },
  mapLinkText: { ...typography.caption, color: colors.primary, fontWeight: '700' },
  mappingEmptyText: { ...typography.small, color: colors.textMuted, lineHeight: 18 },
  mappingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surface,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.border,
    paddingHorizontal: 14,
    paddingVertical: 10,
    marginBottom: 8,
    gap: 8,
  },
  mappingRowPending: { borderColor: '#F0A742' + '60', backgroundColor: '#FEF3C7' + '30' },
  mappingTopicName: { ...typography.caption, fontWeight: '600', color: colors.text, flex: 1 },
  mappingPages: { ...typography.small, color: colors.textMuted },
  mappingPendingTag: { ...typography.small, fontSize: 9, fontWeight: '700', color: '#92640A' },

  topicRow: { flexDirection: 'row', marginTop: 10, gap: 8 },
  topicBullet: { color: colors.primary, fontSize: 16, lineHeight: 22 },
  topicName: { ...typography.body, fontWeight: '600', color: colors.text },
  topicDescription: { ...typography.small, color: colors.textMuted, marginTop: 2 },

  questionCard: {
    backgroundColor: colors.surface,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 14,
    marginTop: 10,
  },
  questionCardPending: { borderColor: '#F0A742' + '60', backgroundColor: '#FEF3C7' + '30' },
  questionText: { ...typography.body, color: colors.text, lineHeight: 22 },
  optionsList: { marginTop: 8, gap: 4 },
  optionText: { ...typography.small, color: colors.textMuted },
  answerText: { ...typography.small, color: colors.success, fontWeight: '700', marginTop: 8 },
  yearText: { ...typography.small, color: colors.textLight, marginTop: 4 },
  questionMetaRow: { flexDirection: 'row', alignItems: 'center', gap: 10, marginTop: 6 },
});
