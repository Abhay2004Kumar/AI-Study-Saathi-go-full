// Client-side tree shape for the syllabus editor. `_key` is a local-only
// identity for React lists and in-place edits; `id` is only present once a
// node has actually been persisted by the backend. Keeping these separate
// means a freshly AI-extracted or manually-typed node (no `id` yet) can't be
// mistaken for an existing one when the tree is submitted.
export interface SubtopicNode {
  _key: string;
  id?: string;
  name: string;
  description?: string;
}

export interface TopicNode {
  _key: string;
  id?: string;
  name: string;
  description?: string;
  subtopics: SubtopicNode[];
}

export interface SubjectNode {
  _key: string;
  id?: string;
  name: string;
  description?: string;
  topics: TopicNode[];
}

let keyCounter = 0;
export function genKey(): string {
  keyCounter += 1;
  return `k${Date.now()}_${keyCounter}`;
}

export function emptySubtopic(): SubtopicNode {
  return { _key: genKey(), name: '' };
}

export function emptyTopic(): TopicNode {
  return { _key: genKey(), name: '', subtopics: [] };
}

export function emptySubject(): SubjectNode {
  return { _key: genKey(), name: '', topics: [] };
}

/** Adds local `_key`s to a tree fetched from the API or suggested by AI extraction. */
export function fromApiSubjects(subjects: any[]): SubjectNode[] {
  return (subjects || []).map((s) => ({
    _key: genKey(),
    id: s.id,
    name: s.name || '',
    description: s.description || undefined,
    topics: (s.topics || []).map((t: any) => ({
      _key: genKey(),
      id: t.id,
      name: t.name || '',
      description: t.description || undefined,
      subtopics: (t.subtopics || []).map((st: any) => ({
        _key: genKey(),
        id: st.id,
        name: st.name || '',
        description: st.description || undefined,
      })),
    })),
  }));
}

/** Strips local-only fields and empty nodes before submitting to the API. */
export function toApiSubjects(subjects: SubjectNode[]) {
  return subjects
    .filter((s) => s.name.trim())
    .map((s) => ({
      ...(s.id ? { id: s.id } : {}),
      name: s.name.trim(),
      ...(s.description?.trim() ? { description: s.description.trim() } : {}),
      topics: s.topics
        .filter((t) => t.name.trim())
        .map((t) => ({
          ...(t.id ? { id: t.id } : {}),
          name: t.name.trim(),
          ...(t.description?.trim() ? { description: t.description.trim() } : {}),
          subtopics: t.subtopics
            .filter((st) => st.name.trim())
            .map((st) => ({
              ...(st.id ? { id: st.id } : {}),
              name: st.name.trim(),
              ...(st.description?.trim() ? { description: st.description.trim() } : {}),
            })),
        })),
    }));
}

export function updateSubject(subjects: SubjectNode[], key: string, patch: Partial<SubjectNode>): SubjectNode[] {
  return subjects.map((s) => (s._key === key ? { ...s, ...patch } : s));
}

export function removeSubject(subjects: SubjectNode[], key: string): SubjectNode[] {
  return subjects.filter((s) => s._key !== key);
}

export function updateTopic(
  subjects: SubjectNode[],
  subjectKey: string,
  topicKey: string,
  patch: Partial<TopicNode>
): SubjectNode[] {
  return subjects.map((s) =>
    s._key !== subjectKey ? s : { ...s, topics: s.topics.map((t) => (t._key === topicKey ? { ...t, ...patch } : t)) }
  );
}

export function removeTopic(subjects: SubjectNode[], subjectKey: string, topicKey: string): SubjectNode[] {
  return subjects.map((s) =>
    s._key !== subjectKey ? s : { ...s, topics: s.topics.filter((t) => t._key !== topicKey) }
  );
}

export function updateSubtopic(
  subjects: SubjectNode[],
  subjectKey: string,
  topicKey: string,
  subtopicKey: string,
  patch: Partial<SubtopicNode>
): SubjectNode[] {
  return subjects.map((s) =>
    s._key !== subjectKey
      ? s
      : {
          ...s,
          topics: s.topics.map((t) =>
            t._key !== topicKey
              ? t
              : { ...t, subtopics: t.subtopics.map((st) => (st._key === subtopicKey ? { ...st, ...patch } : st)) }
          ),
        }
  );
}

export function removeSubtopic(
  subjects: SubjectNode[],
  subjectKey: string,
  topicKey: string,
  subtopicKey: string
): SubjectNode[] {
  return subjects.map((s) =>
    s._key !== subjectKey
      ? s
      : {
          ...s,
          topics: s.topics.map((t) =>
            t._key !== topicKey ? t : { ...t, subtopics: t.subtopics.filter((st) => st._key !== subtopicKey) }
          ),
        }
  );
}
