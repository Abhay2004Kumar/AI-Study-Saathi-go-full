import apiClient from './client';

export const mappingService = {
  /**
   * Everything mapped to one syllabus topic — notes, PYQs, and syllabus
   * resources, each with the page range and review status.
   * @param {string} examId
   * @param {string} topicId
   * @returns {Promise<Object>}
   */
  async getTopicResources(examId, topicId) {
    const response = await apiClient.get(`/exams/${examId}/topics/${topicId}/resources`);
    return response.data;
  },

  /**
   * A document's own mappings onto its exam's syllabus.
   * @param {string} documentId
   * @returns {Promise<Object>}
   */
  async getDocumentMappings(documentId) {
    const response = await apiClient.get(`/documents/${documentId}/mappings`);
    return response.data;
  },

  /**
   * (Re)generates a document's mappings onto its exam's syllabus.
   * @param {string} documentId
   * @returns {Promise<Object>}
   */
  async mapDocumentTopics(documentId) {
    const response = await apiClient.post(`/documents/${documentId}/map-topics`);
    return response.data;
  },

  /**
   * Confirms or rejects a mapping — the review step for low-confidence AI guesses.
   * @param {string} mappingId
   * @param {'CONFIRMED'|'REJECTED'} status
   * @returns {Promise<Object>}
   */
  async updateMappingStatus(mappingId, status) {
    const response = await apiClient.patch(`/mappings/${mappingId}`, { status });
    return response.data;
  },
};
