import apiClient from './client';

export const flashcardService = {
  /**
   * All flashcards in a topic's deck, including their spaced-repetition state.
   * @param {string} examId
   * @param {string} topicId
   * @returns {Promise<Object>}
   */
  async getTopicFlashcards(examId, topicId) {
    const response = await apiClient.get(`/exams/${examId}/topics/${topicId}/flashcards`);
    return response.data;
  },

  /**
   * Generates and adds new flashcards to a topic's deck — additive, cards
   * already mid-schedule are untouched.
   * @param {string} examId
   * @param {string} topicId
   * @param {Object} [options]
   * @param {number} [options.numberOfCards]
   * @param {string} [options.subtopicId]
   * @returns {Promise<Object>}
   */
  async generateTopicFlashcards(examId, topicId, options = {}) {
    const response = await apiClient.post(`/exams/${examId}/topics/${topicId}/flashcards`, options);
    return response.data;
  },

  /**
   * Cards due for review right now across the whole exam, soonest first.
   * @param {string} examId
   * @param {number} [limit]
   * @returns {Promise<Object>}
   */
  async getDueFlashcards(examId, limit) {
    const response = await apiClient.get(`/exams/${examId}/flashcards/due`, { params: limit ? { limit } : {} });
    return response.data;
  },

  /**
   * Grades a review. Scheduling (SM-2) happens server-side, deterministically.
   * @param {string} cardId
   * @param {'AGAIN'|'HARD'|'GOOD'|'EASY'} grade
   * @returns {Promise<Object>}
   */
  async reviewFlashcard(cardId, grade) {
    const response = await apiClient.post(`/flashcards/${cardId}/review`, { grade });
    return response.data;
  },
};
