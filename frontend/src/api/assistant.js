import apiClient from './client';

export const assistantService = {
  /**
   * This exam's existing conversation with the AI Study Assistant, or null
   * if nothing has been sent yet.
   * @param {string} examId
   * @returns {Promise<Object>}
   */
  async getSession(examId) {
    const response = await apiClient.get(`/exams/${examId}/assistant`);
    return response.data;
  },

  /**
   * Sends a message to the exam-scoped assistant. It answers using tools
   * over the student's own mastery, study plan, PYQ importance, flashcards,
   * and materials — never free-associating about their progress.
   * @param {string} examId
   * @param {string} message
   * @returns {Promise<Object>}
   */
  async sendMessage(examId, message) {
    const response = await apiClient.post(`/exams/${examId}/assistant/message`, { message });
    return response.data;
  },
};
