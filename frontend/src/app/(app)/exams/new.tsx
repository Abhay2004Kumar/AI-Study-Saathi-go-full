import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Alert, TextInput as RNTextInput } from 'react-native';
import { useRouter } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { examService } from '../../../api/exams';
import AppTextInput from '../../../components/AppTextInput';
import AppButton from '../../../components/AppButton';
import SyllabusTree from '../../../components/SyllabusTree';
import KeyboardAwareScrollView from '../../../components/KeyboardAwareScrollView';
import { type SubjectNode, fromApiSubjects, toApiSubjects } from '../../../utils/syllabusTree';
import { colors } from '../../../theme/colors';
import { typography } from '../../../theme/typography';

export default function NewExamScreen() {
  const router = useRouter();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [examDate, setExamDate] = useState('');
  const [hoursPerDay, setHoursPerDay] = useState('');
  const [subjects, setSubjects] = useState<SubjectNode[]>([]);
  const [saving, setSaving] = useState(false);

  const [mode, setMode] = useState<'manual' | 'extract'>('manual');
  const [extractText, setExtractText] = useState('');
  const [extracting, setExtracting] = useState(false);

  const handleExtract = async () => {
    if (!extractText.trim()) {
      Alert.alert('Nothing to Extract', 'Paste your syllabus text first');
      return;
    }
    try {
      setExtracting(true);
      const res: any = await examService.extractSyllabus(extractText.trim());
      const suggested = fromApiSubjects(res.data?.subjects || []);
      if (suggested.length === 0) {
        Alert.alert('No Structure Found', "Couldn't extract a syllabus structure from that text. Try adding it manually instead.");
        return;
      }
      setSubjects((prev) => [...prev, ...suggested]);
      setMode('manual');
      Alert.alert('Suggested Structure Added', 'Review and edit the extracted subjects/topics below, then create your exam.');
    } catch (error: any) {
      Alert.alert('Extraction Failed', error.response?.data?.message || error.message);
    } finally {
      setExtracting(false);
    }
  };

  const handleCreate = async () => {
    if (!name.trim()) {
      Alert.alert('Missing Name', 'Please give this exam a name');
      return;
    }
    const parsedHours = hoursPerDay ? Number(hoursPerDay) : undefined;
    if (hoursPerDay && (Number.isNaN(parsedHours) || parsedHours! <= 0)) {
      Alert.alert('Invalid Hours', 'Available hours per day must be a positive number');
      return;
    }
    if (examDate && Number.isNaN(Date.parse(examDate))) {
      Alert.alert('Invalid Date', 'Exam date should look like YYYY-MM-DD');
      return;
    }

    try {
      setSaving(true);
      await examService.createExam({
        name: name.trim(),
        description: description.trim() || undefined,
        examDate: examDate.trim() || undefined,
        availableHoursPerDay: parsedHours,
        subjects: toApiSubjects(subjects),
      });
      router.replace('/(app)/exams' as any);
    } catch (error: any) {
      Alert.alert('Could Not Create Exam', error.response?.data?.message || error.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()} style={styles.headerBack}>
          <Text style={styles.headerBackText}>‹</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>New Exam</Text>
        <View style={{ width: 36 }} />
      </View>

      <KeyboardAwareScrollView contentContainerStyle={styles.scrollContent}>
        <AppTextInput label="Exam Name" placeholder="e.g. STET Computer Science" value={name} onChangeText={setName} icon="🎯" />
        <AppTextInput
          label="Description (optional)"
          placeholder="A short note about this exam"
          value={description}
          onChangeText={setDescription}
          icon="📝"
        />
        <View style={styles.row}>
          <View style={styles.rowItem}>
            <AppTextInput label="Exam Date (optional)" placeholder="YYYY-MM-DD" value={examDate} onChangeText={setExamDate} icon="📅" />
          </View>
          <View style={styles.rowItem}>
            <AppTextInput
              label="Hours/Day (optional)"
              placeholder="e.g. 3"
              value={hoursPerDay}
              onChangeText={setHoursPerDay}
              keyboardType="numeric"
              icon="⏱"
            />
          </View>
        </View>

        <Text style={styles.sectionTitle}>Syllabus</Text>

        <View style={styles.modeToggle}>
          <TouchableOpacity
            style={[styles.modeBtn, mode === 'manual' && styles.modeBtnActive]}
            onPress={() => setMode('manual')}
            activeOpacity={0.8}
          >
            <Text style={[styles.modeBtnText, mode === 'manual' && styles.modeBtnTextActive]}>Build Manually</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.modeBtn, mode === 'extract' && styles.modeBtnActive]}
            onPress={() => setMode('extract')}
            activeOpacity={0.8}
          >
            <Text style={[styles.modeBtnText, mode === 'extract' && styles.modeBtnTextActive]}>✨ AI Extract from Text</Text>
          </TouchableOpacity>
        </View>

        {mode === 'extract' && (
          <View style={styles.extractCard}>
            <Text style={styles.extractLabel}>Paste your syllabus text</Text>
            <RNTextInput
              style={styles.extractInput}
              value={extractText}
              onChangeText={setExtractText}
              placeholder="e.g. Unit 1: Process Management - scheduling, states...&#10;Unit 2: Deadlocks - detection, prevention..."
              placeholderTextColor={colors.textLight}
              multiline
              textAlignVertical="top"
            />
            <AppButton
              title={extracting ? 'Extracting...' : '✨ Extract Structure'}
              onPress={handleExtract}
              loading={extracting}
              variant="secondary"
            />
          </View>
        )}

        <SyllabusTree subjects={subjects} onChange={setSubjects} />

        <AppButton
          title={saving ? 'Creating...' : 'Create Exam'}
          onPress={handleCreate}
          loading={saving}
          style={styles.createBtn}
        />
      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1, backgroundColor: colors.background },
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
  headerTitle: { ...typography.subtitle, color: colors.text },

  scrollContent: {
    paddingHorizontal: 20,
    paddingBottom: 40,
    width: '100%',
    maxWidth: 520,
    alignSelf: 'center',
  },

  row: { flexDirection: 'row', gap: 12 },
  rowItem: { flex: 1 },

  sectionTitle: { ...typography.subtitle, color: colors.text, marginTop: 8, marginBottom: 12 },

  modeToggle: {
    flexDirection: 'row',
    backgroundColor: colors.surfaceMid,
    borderRadius: 14,
    padding: 4,
    marginBottom: 16,
  },
  modeBtn: { flex: 1, paddingVertical: 10, borderRadius: 10, alignItems: 'center' },
  modeBtnActive: { backgroundColor: colors.surface },
  modeBtnText: { ...typography.caption, fontWeight: '700', color: colors.textMuted },
  modeBtnTextActive: { color: colors.primary },

  extractCard: {
    backgroundColor: colors.surface,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 16,
    marginBottom: 16,
  },
  extractLabel: { ...typography.caption, color: colors.textMuted, marginBottom: 8 },
  extractInput: {
    ...typography.body,
    color: colors.text,
    backgroundColor: colors.surfaceMid,
    borderWidth: 1.5,
    borderColor: colors.border,
    borderRadius: 14,
    padding: 14,
    minHeight: 120,
    marginBottom: 12,
  },

  createBtn: { marginTop: 12 },
});
