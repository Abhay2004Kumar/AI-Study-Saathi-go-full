import React, { useRef } from 'react';
import { View, Text, TextInput, TouchableOpacity, StyleSheet } from 'react-native';
import { colors } from '../theme/colors';
import { typography } from '../theme/typography';
import { useScrollIntoView } from './KeyboardAwareScrollView';
import {
  type SubjectNode,
  type TopicNode,
  type SubtopicNode,
  emptySubject,
  emptyTopic,
  emptySubtopic,
  updateSubject,
  removeSubject,
  updateTopic,
  removeTopic,
  updateSubtopic,
  removeSubtopic,
} from '../utils/syllabusTree';

interface SyllabusTreeProps {
  subjects: SubjectNode[];
  onChange?: (subjects: SubjectNode[]) => void;
  readOnly?: boolean;
  /** When set (and readOnly), each topic row becomes tappable — e.g. to browse its mapped resources. */
  onTopicPress?: (topic: TopicNode) => void;
}

// A compact row input (no label/icon chrome) that still lifts itself above
// the keyboard when nested inside a KeyboardAwareScrollView, same as
// AppTextInput — the tree can get tall enough that a lower row would
// otherwise end up hidden behind the keyboard.
function TreeInput({
  value,
  onChangeText,
  placeholder,
  style,
}: {
  value: string;
  onChangeText: (text: string) => void;
  placeholder: string;
  style?: any;
}) {
  const ref = useRef<TextInput>(null);
  const scrollIntoView = useScrollIntoView();

  return (
    <TextInput
      ref={ref}
      style={[styles.input, style]}
      value={value}
      onChangeText={onChangeText}
      placeholder={placeholder}
      placeholderTextColor={colors.textLight}
      onFocus={() => scrollIntoView?.(ref.current)}
    />
  );
}

export default function SyllabusTree({ subjects, onChange, readOnly = false, onTopicPress }: SyllabusTreeProps) {
  const emit = (next: SubjectNode[]) => onChange?.(next);

  return (
    <View style={styles.container}>
      {subjects.length === 0 && (
        <Text style={styles.emptyText}>
          {readOnly ? 'No syllabus added yet.' : 'No subjects yet — add one to start building the syllabus.'}
        </Text>
      )}

      {subjects.map((subject) => (
        <SubjectRow
          key={subject._key}
          subject={subject}
          readOnly={readOnly}
          onTopicPress={onTopicPress}
          onChangeName={(name) => emit(updateSubject(subjects, subject._key, { name }))}
          onRemove={() => emit(removeSubject(subjects, subject._key))}
          onAddTopic={() => emit(updateSubject(subjects, subject._key, { topics: [...subject.topics, emptyTopic()] }))}
          onChangeTopicName={(topicKey, name) => emit(updateTopic(subjects, subject._key, topicKey, { name }))}
          onRemoveTopic={(topicKey) => emit(removeTopic(subjects, subject._key, topicKey))}
          onAddSubtopic={(topicKey) => {
            const topic = subject.topics.find((t) => t._key === topicKey);
            if (!topic) return;
            emit(updateTopic(subjects, subject._key, topicKey, { subtopics: [...topic.subtopics, emptySubtopic()] }));
          }}
          onChangeSubtopicName={(topicKey, subtopicKey, name) =>
            emit(updateSubtopic(subjects, subject._key, topicKey, subtopicKey, { name }))
          }
          onRemoveSubtopic={(topicKey, subtopicKey) =>
            emit(removeSubtopic(subjects, subject._key, topicKey, subtopicKey))
          }
        />
      ))}

      {!readOnly && (
        <TouchableOpacity
          style={styles.addSubjectBtn}
          onPress={() => emit([...subjects, emptySubject()])}
          activeOpacity={0.8}
        >
          <Text style={styles.addSubjectBtnText}>+ Add Subject</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

function SubjectRow({
  subject,
  readOnly,
  onTopicPress,
  onChangeName,
  onRemove,
  onAddTopic,
  onChangeTopicName,
  onRemoveTopic,
  onAddSubtopic,
  onChangeSubtopicName,
  onRemoveSubtopic,
}: {
  subject: SubjectNode;
  readOnly: boolean;
  onTopicPress?: (topic: TopicNode) => void;
  onChangeName: (name: string) => void;
  onRemove: () => void;
  onAddTopic: () => void;
  onChangeTopicName: (topicKey: string, name: string) => void;
  onRemoveTopic: (topicKey: string) => void;
  onAddSubtopic: (topicKey: string) => void;
  onChangeSubtopicName: (topicKey: string, subtopicKey: string, name: string) => void;
  onRemoveSubtopic: (topicKey: string, subtopicKey: string) => void;
}) {
  return (
    <View style={styles.subjectCard}>
      <View style={styles.rowHeader}>
        {readOnly ? (
          <Text style={styles.subjectText} numberOfLines={2}>📘 {subject.name}</Text>
        ) : (
          <>
            <Text style={styles.rowIcon}>📘</Text>
            <TreeInput value={subject.name} onChangeText={onChangeName} placeholder="Subject name" style={styles.subjectInput} />
            <TouchableOpacity onPress={onRemove} style={styles.removeBtn} hitSlop={8}>
              <Text style={styles.removeBtnText}>×</Text>
            </TouchableOpacity>
          </>
        )}
      </View>

      {subject.topics.map((topic: TopicNode) => (
        <View key={topic._key} style={styles.topicBlock}>
          <View style={styles.rowHeader}>
            {readOnly ? (
              onTopicPress ? (
                <TouchableOpacity style={styles.topicPressable} onPress={() => onTopicPress(topic)} activeOpacity={0.7}>
                  <Text style={styles.topicText} numberOfLines={2}>• {topic.name}</Text>
                  <Text style={styles.topicChevron}>›</Text>
                </TouchableOpacity>
              ) : (
                <Text style={styles.topicText} numberOfLines={2}>• {topic.name}</Text>
              )
            ) : (
              <>
                <Text style={styles.rowIcon}>•</Text>
                <TreeInput
                  value={topic.name}
                  onChangeText={(name) => onChangeTopicName(topic._key, name)}
                  placeholder="Topic name"
                  style={styles.topicInput}
                />
                <TouchableOpacity onPress={() => onRemoveTopic(topic._key)} style={styles.removeBtn} hitSlop={8}>
                  <Text style={styles.removeBtnText}>×</Text>
                </TouchableOpacity>
              </>
            )}
          </View>

          {topic.subtopics.map((subtopic: SubtopicNode) => (
            <View key={subtopic._key} style={styles.rowHeader}>
              {readOnly ? (
                <Text style={styles.subtopicText} numberOfLines={2}>‣ {subtopic.name}</Text>
              ) : (
                <>
                  <Text style={styles.rowIconSmall}>‣</Text>
                  <TreeInput
                    value={subtopic.name}
                    onChangeText={(name) => onChangeSubtopicName(topic._key, subtopic._key, name)}
                    placeholder="Subtopic name"
                    style={styles.subtopicInput}
                  />
                  <TouchableOpacity
                    onPress={() => onRemoveSubtopic(topic._key, subtopic._key)}
                    style={styles.removeBtn}
                    hitSlop={8}
                  >
                    <Text style={styles.removeBtnText}>×</Text>
                  </TouchableOpacity>
                </>
              )}
            </View>
          ))}

          {!readOnly && (
            <TouchableOpacity onPress={() => onAddSubtopic(topic._key)} style={styles.addLink}>
              <Text style={styles.addLinkText}>+ Add Subtopic</Text>
            </TouchableOpacity>
          )}
        </View>
      ))}

      {!readOnly && (
        <TouchableOpacity onPress={onAddTopic} style={styles.addLink}>
          <Text style={styles.addLinkText}>+ Add Topic</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { width: '100%' },
  emptyText: {
    ...typography.body,
    color: colors.textMuted,
    textAlign: 'center',
    paddingVertical: 16,
  },
  subjectCard: {
    backgroundColor: colors.surface,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 14,
    marginBottom: 12,
  },
  rowHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 6,
  },
  rowIcon: { fontSize: 16, marginRight: 8 },
  rowIconSmall: { fontSize: 13, marginRight: 8, marginLeft: 28, color: colors.textMuted },
  subjectText: { ...typography.subtitle, fontSize: 16, color: colors.text, flex: 1 },
  topicText: { ...typography.body, fontWeight: '600', color: colors.text, flex: 1, marginLeft: 24 },
  topicPressable: { flexDirection: 'row', alignItems: 'center', flex: 1 },
  topicChevron: { fontSize: 18, color: colors.textLight, marginLeft: 4 },
  subtopicText: { ...typography.caption, color: colors.textMuted, flex: 1 },
  input: {
    flex: 1,
    ...typography.body,
    color: colors.text,
    backgroundColor: colors.surfaceMid,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  subjectInput: { fontWeight: '700' },
  topicInput: { marginLeft: 24 },
  subtopicInput: {},
  removeBtn: {
    marginLeft: 8,
    width: 26,
    height: 26,
    borderRadius: 13,
    backgroundColor: colors.surfaceWarm,
    justifyContent: 'center',
    alignItems: 'center',
  },
  removeBtnText: { color: colors.error, fontSize: 16, fontWeight: '700', lineHeight: 18 },
  topicBlock: {
    marginLeft: 12,
    marginTop: 6,
    paddingLeft: 8,
    borderLeftWidth: 2,
    borderLeftColor: colors.divider,
  },
  addLink: {
    alignSelf: 'flex-start',
    marginTop: 2,
    marginBottom: 4,
    marginLeft: 24,
    paddingVertical: 4,
  },
  addLinkText: { ...typography.caption, color: colors.primary, fontWeight: '700' },
  addSubjectBtn: {
    backgroundColor: colors.primaryLight,
    borderRadius: 14,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 4,
  },
  addSubjectBtnText: { ...typography.body, fontWeight: '700', color: colors.primaryDark },
});
