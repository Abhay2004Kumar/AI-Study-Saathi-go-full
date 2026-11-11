import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator, RefreshControl, TouchableOpacity } from 'react-native';
import { useRouter, useFocusEffect } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { examService } from '../../../api/exams';
import { colors } from '../../../theme/colors';
import { typography } from '../../../theme/typography';

export default function ExamsListScreen() {
  const router = useRouter();
  const [exams, setExams] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchExams = async () => {
    try {
      const res: any = await examService.getExams();
      setExams(res.data || []);
    } catch (error) {
      console.log('Error fetching exams', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useFocusEffect(
    useCallback(() => {
      fetchExams();
    }, [])
  );

  const onRefresh = () => {
    setRefreshing(true);
    fetchExams();
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return null;
    return new Date(dateStr).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' });
  };

  const renderHeader = () => (
    <View style={styles.headerRow}>
      <View style={styles.headerText}>
        <Text style={styles.title}>My Exams 🎯</Text>
        <Text style={styles.subtitle}>Organize your syllabus and track prep by exam</Text>
      </View>
      <TouchableOpacity style={styles.addBtn} onPress={() => router.push('/(app)/exams/new' as any)} activeOpacity={0.8}>
        <Text style={styles.addBtnText}>+ New</Text>
      </TouchableOpacity>
    </View>
  );

  const renderEmpty = () => {
    if (loading) {
      return (
        <View style={styles.emptyContainer}>
          <ActivityIndicator size="large" color={colors.primary} />
          <Text style={styles.loadingText}>Loading your exams...</Text>
        </View>
      );
    }
    return (
      <View style={styles.emptyContainer}>
        <View style={styles.emptyIllustration}>
          <Text style={styles.emptyEmoji}>🗂️</Text>
        </View>
        <Text style={styles.emptyTitle}>No exams yet</Text>
        <Text style={styles.emptyText}>
          Create an exam and build its syllabus to start organizing your study material around it.
        </Text>
        <TouchableOpacity style={styles.emptyCta} onPress={() => router.push('/(app)/exams/new' as any)} activeOpacity={0.8}>
          <Text style={styles.emptyCtaText}>Create Your First Exam →</Text>
        </TouchableOpacity>
      </View>
    );
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <FlatList
        data={exams}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.listContent}
        ListHeaderComponent={renderHeader}
        ListEmptyComponent={renderEmpty}
        showsVerticalScrollIndicator={false}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={colors.primary} colors={[colors.primary]} />}
        renderItem={({ item }) => (
          <TouchableOpacity
            style={styles.examCard}
            onPress={() => router.push({ pathname: '/(app)/exams/[id]', params: { id: item.id } } as any)}
            activeOpacity={0.85}
          >
            <View style={styles.examIconBox}>
              <Text style={styles.examIconText}>🎯</Text>
            </View>
            <View style={styles.examInfo}>
              <Text style={styles.examName} numberOfLines={1}>{item.name}</Text>
              <Text style={styles.examMeta} numberOfLines={1}>
                {item._count?.subjects || 0} subject{item._count?.subjects === 1 ? '' : 's'}
                {formatDate(item.examDate) ? ` • Exam on ${formatDate(item.examDate)}` : ''}
              </Text>
            </View>
            <Text style={styles.chevron}>›</Text>
          </TouchableOpacity>
        )}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1, backgroundColor: colors.background },
  listContent: { paddingHorizontal: 20, paddingBottom: 40, flexGrow: 1 },

  headerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    paddingTop: 20,
    paddingBottom: 20,
  },
  headerText: { flex: 1, marginRight: 12 },
  title: { ...typography.header, fontSize: 26, color: colors.text },
  subtitle: { ...typography.body, color: colors.textMuted, marginTop: 2 },
  addBtn: {
    backgroundColor: colors.success,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  addBtnText: { ...typography.caption, fontWeight: '700', color: '#FFFFFF' },

  examCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surfaceWarm,
    borderRadius: 18,
    padding: 16,
    marginBottom: 12,
    borderWidth: 1,
    borderColor: colors.border,
  },
  examIconBox: {
    width: 44,
    height: 44,
    borderRadius: 12,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  examIconText: { fontSize: 22 },
  examInfo: { flex: 1 },
  examName: { ...typography.subtitle, fontSize: 16, color: colors.text },
  examMeta: { ...typography.small, color: colors.textMuted, marginTop: 2 },
  chevron: { fontSize: 24, color: colors.textLight, marginLeft: 8 },

  emptyContainer: { flex: 1, alignItems: 'center', paddingTop: 40, paddingHorizontal: 20 },
  emptyIllustration: {
    width: 100,
    height: 100,
    borderRadius: 50,
    backgroundColor: colors.surfaceWarm,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 20,
    borderWidth: 2,
    borderColor: colors.border,
  },
  emptyEmoji: { fontSize: 46 },
  emptyTitle: { ...typography.subtitle, color: colors.text, marginBottom: 10 },
  emptyText: { ...typography.body, color: colors.textMuted, textAlign: 'center', lineHeight: 22, marginBottom: 28 },
  emptyCta: { backgroundColor: colors.primary, borderRadius: 20, paddingHorizontal: 24, paddingVertical: 14 },
  emptyCtaText: { ...typography.body, fontWeight: '700', color: '#FFFFFF' },
  loadingText: { ...typography.body, color: colors.textMuted, marginTop: 16 },
});
