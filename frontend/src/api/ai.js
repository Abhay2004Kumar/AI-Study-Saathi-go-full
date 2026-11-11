import apiClient from './client';

export const aiService = {
  /**
   * Ask a question answered from the user's own uploaded materials (RAG).
   * @param {string} question
   * @param {Object} [filters]
   * @param {string} [filters.examId] - scope retrieval to one exam's resources
   * @param {'SYLLABUS'|'NOTES'|'PYQ'|'GENERAL'} [filters.category]
   * @returns {Promise<Object>} data: { answer, foundInMaterials, sources: [{title, page}] }
   */
  async askQuestion(question, filters = {}) {
    const response = await apiClient.post('/ai/ask', {
      question,
      examId: filters.examId,
      category: filters.category,
    });
    return response.data;
  },
};
