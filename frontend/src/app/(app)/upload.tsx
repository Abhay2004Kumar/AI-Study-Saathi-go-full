import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, Alert, TouchableOpacity, ActivityIndicator, Linking } from 'react-native';
import * as DocumentPicker from 'expo-document-picker';
import * as ImagePicker from 'expo-image-picker';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { documentService } from '../../api/documents';
import { examService } from '../../api/exams';
import AppButton from '../../components/AppButton';
import AppTextInput from '../../components/AppTextInput';
import KeyboardAwareScrollView from '../../components/KeyboardAwareScrollView';
import { colors } from '../../theme/colors';
import { typography } from '../../theme/typography';

const CATEGORIES: { value: 'GENERAL' | 'NOTES' | 'PYQ' | 'SYLLABUS'; label: string; emoji: string }[] = [
  { value: 'GENERAL', label: 'General', emoji: '📄' },
  { value: 'NOTES', label: 'Notes', emoji: '📝' },
  { value: 'PYQ', label: 'Past Papers', emoji: '🗞️' },
  { value: 'SYLLABUS', label: 'Syllabus', emoji: '📘' },
];

// The shape documentService.uploadDocument actually needs — both
// expo-document-picker and (normalized) expo-image-picker results satisfy it.
type PickedFile = { uri: string; name: string; mimeType?: string | null; size?: number };

export default function UploadScreen() {
  const router = useRouter();
  const params = useLocalSearchParams();
  const lockedExamId = typeof params.examId === 'string' ? params.examId : undefined;

  const [title, setTitle] = useState('');
  const [file, setFile] = useState<PickedFile | null>(null);
  const [category, setCategory] = useState<typeof CATEGORIES[number]['value']>('GENERAL');
  const [uploading, setUploading] = useState(false);

  const [exams, setExams] = useState<any[]>([]);
  const [examsLoading, setExamsLoading] = useState(!lockedExamId);
  const [selectedExamId, setSelectedExamId] = useState<string | undefined>(lockedExamId);
  const [lockedExamName, setLockedExamName] = useState<string | null>(null);

  useEffect(() => {
    if (lockedExamId) {
      examService
        .getExam(lockedExamId)
        .then((res: any) => setLockedExamName(res.data?.name || null))
        .catch(() => {});
      return;
    }
    examService
      .getExams()
      .then((res: any) => setExams(res.data || []))
      .catch((err) => console.log('Failed to load exams for upload picker', err))
      .finally(() => setExamsLoading(false));
  }, [lockedExamId]);

  const handlePickDocument = async () => {
    try {
      const result = await DocumentPicker.getDocumentAsync({
        type: ['application/pdf', 'text/plain', 'text/markdown', 'image/jpeg', 'image/png'],
        copyToCacheDirectory: true,
      });
      if (!result.canceled && result.assets && result.assets.length > 0) {
        const pickedFile = result.assets[0];
        setFile(pickedFile);
        if (!title) {
          setTitle(pickedFile.name.split('.').slice(0, -1).join('.'));
        }
      }
    } catch {
      Alert.alert('Error', 'Failed to pick a document');
    }
  };

  const alertPermissionDenied = (what: string) => {
    Alert.alert(
      'Permission Needed',
      `Study Saathi needs ${what} access to do this. You can enable it in your device Settings.`,
      [{ text: 'Cancel', style: 'cancel' }, { text: 'Open Settings', onPress: () => Linking.openSettings() }]
    );
  };

  const applyImagePickerResult = (result: ImagePicker.ImagePickerResult) => {
    if (result.canceled || !result.assets || result.assets.length === 0) return;
    const asset = result.assets[0];
    const pickedFile: PickedFile = {
      uri: asset.uri,
      name: asset.fileName || `photo-${Date.now()}.jpg`,
      mimeType: asset.mimeType || 'image/jpeg',
      size: asset.fileSize,
    };
    setFile(pickedFile);
    if (!title) {
      setTitle(pickedFile.name.split('.').slice(0, -1).join('.') || 'Photo Notes');
    }
  };

  const handleTakePhoto = async () => {
    try {
      const permission = await ImagePicker.requestCameraPermissionsAsync();
      if (!permission.granted) {
        alertPermissionDenied('camera');
        return;
      }
      const result = await ImagePicker.launchCameraAsync({ mediaTypes: ['images'], quality: 0.9 });
      applyImagePickerResult(result);
    } catch {
      Alert.alert('Error', 'Failed to open the camera');
    }
  };

  const handlePickFromGallery = async () => {
    try {
      const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
      if (!permission.granted) {
        alertPermissionDenied('photo library');
        return;
      }
      const result = await ImagePicker.launchImageLibraryAsync({ mediaTypes: ['images'], quality: 0.9 });
      applyImagePickerResult(result);
    } catch {
      Alert.alert('Error', 'Failed to open the photo library');
    }
  };

  const handleUpload = async () => {
    if (!file) { Alert.alert('No File', 'Please select a document first'); return; }
    if (!title) { Alert.alert('No Title', 'Please enter a title for the document'); return; }

    try {
      setUploading(true);
      await documentService.uploadDocument(file, title, { examId: selectedExamId, category });
      Alert.alert('🎉 Uploaded!', 'Your document has been uploaded and is now processing.', [
        { text: 'Go to Dashboard', onPress: () => router.back() }
      ]);
    } catch (error: any) {
      if (error.response?.status === 409) {
        Alert.alert('Already Uploaded', error.response.data.message);
      } else {
        Alert.alert('Upload Failed', error.response?.data?.message || error.message);
      }
    } finally {
      setUploading(false);
    }
  };

  const formatSize = (bytes?: number) => {
    if (!bytes) return '—';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const isImage = (mimeType?: string | null) => mimeType === 'image/jpeg' || mimeType === 'image/png';

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <KeyboardAwareScrollView contentContainerStyle={styles.container}>
        {/* Hero */}
        <View style={styles.hero}>
          <View style={styles.heroIconCircle}>
            <Text style={styles.heroIcon}>📤</Text>
          </View>
          <Text style={styles.title}>Upload Study Material</Text>
          <Text style={styles.subtitle}>Add a PDF, text file, or image to start AI tutoring, quizzes, and flashcards</Text>
        </View>

        <View style={styles.form}>
          <AppTextInput
            label="Document Title"
            placeholder="e.g. DBMS Normalization Notes"
            value={title}
            onChangeText={setTitle}
            editable={!uploading}
            icon="📝"
          />

          {/* Category picker */}
          <Text style={styles.fileLabel}>Category</Text>
          <View style={styles.categoryRow}>
            {CATEGORIES.map((c) => (
              <TouchableOpacity
                key={c.value}
                style={[styles.categoryChip, category === c.value && styles.categoryChipActive]}
                onPress={() => setCategory(c.value)}
                activeOpacity={0.8}
                disabled={uploading}
              >
                <Text style={styles.categoryChipEmoji}>{c.emoji}</Text>
                <Text style={[styles.categoryChipText, category === c.value && styles.categoryChipTextActive]}>
                  {c.label}
                </Text>
              </TouchableOpacity>
            ))}
          </View>

          {/* Exam linking */}
          <Text style={styles.fileLabel}>Link to Exam (optional)</Text>
          {lockedExamId ? (
            <View style={styles.lockedExamPill}>
              <Text style={styles.lockedExamText}>🎯 {lockedExamName || 'Loading…'}</Text>
            </View>
          ) : examsLoading ? (
            <ActivityIndicator color={colors.primary} style={{ marginBottom: 16 }} />
          ) : exams.length === 0 ? (
            <Text style={styles.noExamsText}>Create an exam first to organize uploads under it.</Text>
          ) : (
            <View style={styles.categoryRow}>
              <TouchableOpacity
                style={[styles.categoryChip, !selectedExamId && styles.categoryChipActive]}
                onPress={() => setSelectedExamId(undefined)}
                activeOpacity={0.8}
                disabled={uploading}
              >
                <Text style={[styles.categoryChipText, !selectedExamId && styles.categoryChipTextActive]}>None</Text>
              </TouchableOpacity>
              {exams.map((e) => (
                <TouchableOpacity
                  key={e.id}
                  style={[styles.categoryChip, selectedExamId === e.id && styles.categoryChipActive]}
                  onPress={() => setSelectedExamId(e.id)}
                  activeOpacity={0.8}
                  disabled={uploading}
                >
                  <Text
                    style={[styles.categoryChipText, selectedExamId === e.id && styles.categoryChipTextActive]}
                    numberOfLines={1}
                  >
                    🎯 {e.name}
                  </Text>
                </TouchableOpacity>
              ))}
            </View>
          )}

          {/* File picker */}
          <Text style={styles.fileLabel}>Select File</Text>
          {file ? (
            <View style={styles.fileCard}>
              <Text style={styles.fileEmoji}>
                {file.mimeType === 'application/pdf' ? '📕' : isImage(file.mimeType) ? '🖼️' : '📄'}
              </Text>
              <View style={styles.fileInfo}>
                <Text style={styles.fileName} numberOfLines={2}>{file.name}</Text>
                <Text style={styles.fileSize}>{formatSize(file.size)}</Text>
              </View>
              {!uploading && (
                <TouchableOpacity onPress={() => setFile(null)} style={styles.changeBtn}>
                  <Text style={styles.changeBtnText}>Change</Text>
                </TouchableOpacity>
              )}
            </View>
          ) : (
            <View style={styles.pickZone}>
              <Text style={styles.pickZoneEmoji}>📂</Text>
              <Text style={styles.pickZoneTitle}>Add a document</Text>
              <Text style={styles.pickZoneSubtitle}>PDF, TXT, MD, or a photo of your notes (up to 10MB)</Text>
              <View style={styles.pickActionsRow}>
                <TouchableOpacity style={styles.pickActionBtn} onPress={handleTakePhoto} activeOpacity={0.8}>
                  <Text style={styles.pickActionEmoji}>📷</Text>
                  <Text style={styles.pickActionText}>Camera</Text>
                </TouchableOpacity>
                <TouchableOpacity style={styles.pickActionBtn} onPress={handlePickFromGallery} activeOpacity={0.8}>
                  <Text style={styles.pickActionEmoji}>🖼️</Text>
                  <Text style={styles.pickActionText}>Gallery</Text>
                </TouchableOpacity>
                <TouchableOpacity style={styles.pickActionBtn} onPress={handlePickDocument} activeOpacity={0.8}>
                  <Text style={styles.pickActionEmoji}>📄</Text>
                  <Text style={styles.pickActionText}>Files</Text>
                </TouchableOpacity>
              </View>
            </View>
          )}
          {file && isImage(file.mimeType) && (
            <Text style={styles.imageNote}>
              ℹ️ Photos of handwritten or printed notes are read automatically — the text gets transcribed so it works with tutoring, quizzes, and search just like a typed document.
            </Text>
          )}
        </View>

        <View style={styles.footer}>
          <AppButton
            title={uploading ? 'Uploading...' : 'Upload & Process →'}
            onPress={handleUpload}
            loading={uploading}
            disabled={!file || !title}
            variant="success"
          />
          <TouchableOpacity
            style={styles.cancelBtn}
            onPress={() => router.back()}
            disabled={uploading}
          >
            <Text style={styles.cancelText}>Cancel</Text>
          </TouchableOpacity>
        </View>
      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  container: {
    flexGrow: 1,
    padding: 24,
    // Keeps the form a readable width on tablets and in landscape.
    width: '100%',
    maxWidth: 520,
    alignSelf: 'center',
  },
  hero: {
    alignItems: 'center',
    marginBottom: 32,
    marginTop: 8,
  },
  heroIconCircle: {
    width: 88,
    height: 88,
    borderRadius: 44,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
    borderWidth: 2,
    borderColor: colors.primary + '40',
  },
  heroIcon: { fontSize: 42 },
  title: {
    ...typography.title,
    color: colors.text,
    textAlign: 'center',
    marginBottom: 6,
  },
  subtitle: {
    ...typography.body,
    color: colors.textMuted,
    textAlign: 'center',
    lineHeight: 22,
  },

  form: {
    flex: 1,
  },
  fileLabel: {
    ...typography.caption,
    color: colors.textMuted,
    marginBottom: 8,
    marginLeft: 2,
  },

  categoryRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 20,
  },
  categoryChip: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surfaceMid,
    borderWidth: 1.5,
    borderColor: colors.border,
    borderRadius: 20,
    paddingHorizontal: 14,
    paddingVertical: 8,
    gap: 6,
    maxWidth: '100%',
  },
  categoryChipActive: {
    backgroundColor: colors.primaryLight,
    borderColor: colors.primary,
  },
  categoryChipEmoji: { fontSize: 14 },
  categoryChipText: { ...typography.caption, fontWeight: '600', color: colors.textMuted },
  categoryChipTextActive: { color: colors.primaryDark, fontWeight: '700' },

  lockedExamPill: {
    alignSelf: 'flex-start',
    backgroundColor: colors.primaryLight,
    borderRadius: 20,
    paddingHorizontal: 14,
    paddingVertical: 8,
    marginBottom: 20,
  },
  lockedExamText: { ...typography.caption, fontWeight: '700', color: colors.primaryDark },
  noExamsText: { ...typography.small, color: colors.textMuted, marginBottom: 20 },

  pickZone: {
    backgroundColor: colors.surfaceWarm,
    borderWidth: 2,
    borderColor: colors.border,
    borderStyle: 'dashed' as const,
    borderRadius: 18,
    padding: 32,
    alignItems: 'center',
  },
  pickZoneEmoji: { fontSize: 40, marginBottom: 12 },
  pickZoneTitle: { ...typography.subtitle, fontSize: 16, color: colors.text, marginBottom: 4 },
  pickZoneSubtitle: { ...typography.small, color: colors.textMuted, textAlign: 'center' },
  pickActionsRow: { flexDirection: 'row', gap: 10, marginTop: 20, width: '100%' },
  pickActionBtn: {
    flex: 1,
    backgroundColor: colors.surface,
    borderWidth: 1.5,
    borderColor: colors.border,
    borderRadius: 14,
    paddingVertical: 14,
    alignItems: 'center',
    gap: 4,
  },
  pickActionEmoji: { fontSize: 22 },
  pickActionText: { ...typography.caption, fontWeight: '700', color: colors.text },

  fileCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surface,
    borderRadius: 16,
    padding: 16,
    borderWidth: 1.5,
    borderColor: colors.success + '60',
    gap: 12,
  },
  fileEmoji: { fontSize: 36 },
  fileInfo: { flex: 1 },
  fileName: { ...typography.body, fontWeight: '600', color: colors.text, marginBottom: 2 },
  fileSize: { ...typography.small, color: colors.textMuted },
  changeBtn: {
    backgroundColor: colors.surfaceWarm,
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderWidth: 1,
    borderColor: colors.border,
  },
  changeBtnText: { ...typography.small, fontWeight: '700', color: colors.text },
  imageNote: {
    ...typography.small,
    color: colors.textMuted,
    marginTop: 10,
    lineHeight: 18,
  },

  footer: {
    marginTop: 20,
  },
  cancelBtn: {
    alignItems: 'center',
    paddingVertical: 12,
    marginTop: 4,
  },
  cancelText: {
    ...typography.body,
    color: colors.textMuted,
    fontWeight: '600',
  },
});
