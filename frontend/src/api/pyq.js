import apiClient from './client';

export const pyqService = {
  /**
   * A topic's PYQ intelligence: mapped questions plus a deterministic
   * historical-importance score (frequency + recency + year coverage).
   * @param {string} examId
   * @param {string} topicId
   * @returns {Promise<Object>}
   */
  async getTopicPyqAnalysis(examId, topicId) {
    const response = await apiClient.get(`/exams/${examId}/topics/${topicId}/pyq-analysis`);
    return response.data;
  },

  /**
   * A document's own extracted, topic-mapped PYQ questions.
   * @param {string} documentId
   * @returns {Promise<Object>}
   */
  async getDocumentQuestions(documentId) {
    const response = await apiClient.get(`/documents/${documentId}/questions`);
    return response.data;
  },

  /**
   * (Re)generates a PYQ document's question-level topic mappings.
   * @param {string} documentId
   * @returns {Promise<Object>}
   */
  async analyzePyqDocument(documentId) {
    const response = await apiClient.post(`/documents/${documentId}/analyze-pyq`);
    return response.data;
  },

  /**
   * Confirms or rejects a question's topic assignment — the review step for
   * low-confidence AI guesses.
   * @param {string} questionId
   * @param {'CONFIRMED'|'REJECTED'} status
   * @returns {Promise<Object>}
   */
  async updateQuestionStatus(questionId, status) {
    const response = await apiClient.patch(`/pyq-questions/${questionId}`, { status });
    return response.data;
  },
};
