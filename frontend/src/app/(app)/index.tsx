import React, { useState, useCallback } from 'react';
import {
  View, Text, StyleSheet, FlatList,
  ActivityIndicator, RefreshControl, Alert, TouchableOpacity
} from 'react-native';
import { useRouter, useFocusEffect } from 'expo-router';
import { SafeAreaView, useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuth } from '../../context/AuthContext';
import { documentService } from '../../api/documents';
import AppCard from '../../components/AppCard';
import { colors } from '../../theme/colors';
import { typography } from '../../theme/typography';

export default function DashboardScreen() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const insets = useSafeAreaInsets();

  const [documents, setDocuments] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchDocuments = async () => {
    try {
      const res: any = await documentService.getDocuments();
      setDocuments(res.data || []);
    } catch (error) {
      console.log('Error fetching documents', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleDelete = (id: string, title: string) => {
    Alert.alert(
      'Delete Document',
      `Are you sure you want to delete "${title}"? This cannot be undone.`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await documentService.deleteDocument(id);
              fetchDocuments();
            } catch (error: any) {
              Alert.alert('Error', error.response?.data?.message || 'Failed to delete document');
            }
          }
        }
      ]
    );
  };

  const handleRetry = async (id: string) => {
    try {
      await documentService.retryDocument(id);
      fetchDocuments();
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.message || 'Failed to retry processing');
    }
  };

  useFocusEffect(
    useCallback(() => {
      fetchDocuments();
    }, [])
  );

  const onRefresh = () => {
    setRefreshing(true);
    fetchDocuments();
  };

  // Stats
  const readyCount = documents.filter(d => d.processingStatus === 'READY').length;
  const totalCount = documents.length;

  const renderHeader = () => (
    <View>
      {/* Greeting Header */}
      <View style={styles.greetingRow}>
        <View style={styles.greetingText}>
          <Text style={styles.greeting} numberOfLines={1} adjustsFontSizeToFit minimumFontScale={0.7}>
            Hi {user?.name?.split(' ')[0] || 'Student'}, 👋
          </Text>
          <Text style={styles.greetingSubtitle} numberOfLines={1}>Ready to study today?</Text>
        </View>
        <TouchableOpacity onPress={logout} style={styles.avatarCircle}>
          <Text style={styles.avatarText}>{(user?.name?.[0] || 'S').toUpperCase()}</Text>
        </TouchableOpacity>
      </View>

      {/* Stats Banner */}
      {totalCount > 0 && (
        <View style={styles.statsBanner}>
          <View style={styles.statItem}>
            <Text style={styles.statEmoji}>📄</Text>
            <View style={styles.statText}>
              <Text style={styles.statValue}>{totalCount}</Text>
              <Text style={styles.statLabel} numberOfLines={2}>Documents</Text>
            </View>
          </View>
          <View style={styles.statDivider} />
          <View style={styles.statItem}>
            <Text style={styles.statEmoji}>✅</Text>
            <View style={styles.statText}>
              <Text style={styles.statValue}>{readyCount}</Text>
              <Text style={styles.statLabel} numberOfLines={2}>Ready to Study</Text>
            </View>
          </View>
          <View style={styles.statDivider} />
          <View style={styles.statItem}>
            <Text style={styles.statEmoji}>🔥</Text>
            <View style={styles.statText}>
              <Text style={styles.statValue}>0</Text>
              <Text style={styles.statLabel} numberOfLines={2}>Day Streak</Text>
            </View>
          </View>
        </View>
      )}

      {/* Exams entry point */}
      <TouchableOpacity
        style={styles.examsBanner}
        onPress={() => router.push('/(app)/exams' as any)}
        activeOpacity={0.85}
      >
        <View style={styles.examsBannerIcon}>
          <Text style={styles.examsBannerEmoji}>🎯</Text>
        </View>
        <View style={styles.examsBannerText}>
          <Text style={styles.examsBannerTitle}>My Exams</Text>
          <Text style={styles.examsBannerSubtitle}>Organize your syllabus by exam</Text>
        </View>
        <Text style={styles.chevron}>›</Text>
      </TouchableOpacity>

      {/* Ask AI entry point */}
      <TouchableOpacity
        style={styles.examsBanner}
        onPress={() => router.push('/(app)/ask' as any)}
        activeOpacity={0.85}
      >
        <View style={[styles.examsBannerIcon, { backgroundColor: colors.successLight }]}>
          <Text style={styles.examsBannerEmoji}>✨</Text>
        </View>
        <View style={styles.examsBannerText}>
          <Text style={styles.examsBannerTitle}>Ask AI</Text>
          <Text style={styles.examsBannerSubtitle}>Ask anything, answered from your own materials</Text>
        </View>
        <Text style={styles.chevron}>›</Text>
      </TouchableOpacity>

      {/* Section header */}
      <View style={styles.sectionRow}>
        <Text style={styles.sectionTitle}>My Documents</Text>
        <TouchableOpacity
          style={styles.addBtn}
          onPress={() => router.push('/(app)/upload' as any)}
          activeOpacity={0.8}
        >
          <Text style={styles.addBtnText}>+ Upload</Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  const renderEmptyComponent = () => {
    if (loading) {
      return (
        <View style={styles.emptyContainer}>
          <ActivityIndicator size="large" color={colors.primary} />
          <Text style={styles.loadingText}>Loading your documents...</Text>
        </View>
      );
    }
    return (
      <View style={styles.emptyContainer}>
        <View style={styles.emptyIllustration}>
          <Text style={styles.emptyEmoji}>📭</Text>
        </View>
        <Text style={styles.emptyTitle}>No documents yet</Text>
        <Text style={styles.emptyText}>Upload your first study material to get started with AI tutoring, quizzes, and flashcards.</Text>
        <TouchableOpacity
          style={styles.uploadCta}
          onPress={() => router.push('/(app)/upload' as any)}
          activeOpacity={0.8}
        >
          <Text style={styles.uploadCtaText}>Upload First Document →</Text>
        </TouchableOpacity>
      </View>
    );
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top']}>
      <FlatList
        data={documents}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <AppCard
            title={item.title}
            subtitle={item.fileName}
            status={item.processingStatus}
            category={item.category}
            date={item.createdAt}
            onPress={() => router.push({ pathname: '/(app)/documents/[id]', params: { id: item.id } } as any)}
            actions={item.processingStatus === 'READY' ? [
              {
                label: '🤖 Tutor',
                onPress: () => router.push({
                  pathname: '/(app)/tutoring/[id]',
                  params: { id: item.id, title: item.title }
                } as any)
              },
              {
                label: '🗑',
                variant: 'danger',
                compact: true,
                onPress: () => handleDelete(item.id, item.title)
              }
            ] : item.processingStatus === 'FAILED' ? [
              {
                label: '🔄 Retry',
                onPress: () => handleRetry(item.id)
              },
              {
                label: '🗑 Delete',
                variant: 'danger',
                onPress: () => handleDelete(item.id, item.title)
              }
            ] : [
              {
                label: '🗑 Delete',
                variant: 'danger',
                onPress: () => handleDelete(item.id, item.title)
              }
            ]}
          />
        )}
        contentContainerStyle={[styles.listContent, { paddingBottom: 40 + insets.bottom }]}
        ListHeaderComponent={renderHeader}
        ListEmptyComponent={renderEmptyComponent}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={onRefresh}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        }
        showsVerticalScrollIndicator={false}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  listContent: {
    paddingHorizontal: 20,
    paddingBottom: 40,
    flexGrow: 1,
  },

  // Greeting
  greetingRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: 20,
    paddingBottom: 16,
  },
  greetingText: {
    flex: 1,
  },
  greeting: {
    ...typography.header,
    fontSize: 26,
    color: colors.text,
  },
  greetingSubtitle: {
    ...typography.body,
    color: colors.textMuted,
    marginTop: 2,
  },
  avatarCircle: {
    width: 46,
    height: 46,
    borderRadius: 23,
    backgroundColor: colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
    marginLeft: 12,
  },
  avatarText: {
    ...typography.subtitle,
    color: '#FFFFFF',
    fontSize: 18,
  },

  // Stats Banner
  statsBanner: {
    backgroundColor: colors.primary,
    borderRadius: 20,
    paddingVertical: 18,
    paddingHorizontal: 12,
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 24,
    shadowColor: colors.primaryDark,
    shadowOffset: { width: 0, height: 6 },
    shadowOpacity: 0.25,
    shadowRadius: 10,
    elevation: 6,
  },
  // flex:1 shares the row evenly so the labels wrap instead of being clipped
  // off the edge on narrow (360dp) phones.
  statItem: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
  },
  statText: {
    flexShrink: 1,
  },
  statEmoji: {
    fontSize: 22,
  },
  statValue: {
    ...typography.title,
    fontSize: 20,
    color: '#FFFFFF',
  },
  statLabel: {
    ...typography.small,
    fontSize: 11,
    lineHeight: 15,
    color: 'rgba(255,255,255,0.8)',
  },
  statDivider: {
    width: 1,
    height: 36,
    backgroundColor: 'rgba(255,255,255,0.3)',
  },

  // Exams banner
  examsBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surfaceWarm,
    borderRadius: 18,
    padding: 14,
    marginBottom: 20,
    borderWidth: 1,
    borderColor: colors.border,
  },
  examsBannerIcon: {
    width: 44,
    height: 44,
    borderRadius: 12,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  examsBannerEmoji: { fontSize: 22 },
  examsBannerText: { flex: 1 },
  examsBannerTitle: { ...typography.subtitle, fontSize: 16, color: colors.text },
  examsBannerSubtitle: { ...typography.small, color: colors.textMuted, marginTop: 2 },
  chevron: { fontSize: 24, color: colors.textLight, marginLeft: 8 },

  // Section Row
  sectionRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 14,
  },
  sectionTitle: {
    ...typography.subtitle,
    color: colors.text,
  },
  addBtn: {
    backgroundColor: colors.success,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  addBtnText: {
    ...typography.caption,
    fontWeight: '700',
    color: '#FFFFFF',
  },

  // Empty State
  emptyContainer: {
    flex: 1,
    alignItems: 'center',
    paddingTop: 40,
    paddingHorizontal: 20,
  },
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
  emptyEmoji: {
    fontSize: 46,
  },
  emptyTitle: {
    ...typography.subtitle,
    color: colors.text,
    marginBottom: 10,
  },
  emptyText: {
    ...typography.body,
    color: colors.textMuted,
    textAlign: 'center',
    lineHeight: 22,
    marginBottom: 28,
  },
  uploadCta: {
    backgroundColor: colors.primary,
    borderRadius: 20,
    paddingHorizontal: 24,
    paddingVertical: 14,
  },
  uploadCtaText: {
    ...typography.body,
    fontWeight: '700',
    color: '#FFFFFF',
  },
  loadingText: {
    ...typography.body,
    color: colors.textMuted,
    marginTop: 16,
  },
});
