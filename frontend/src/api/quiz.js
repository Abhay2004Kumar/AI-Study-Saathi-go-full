import apiClient from './client';

export const quizService = {
  /**
   * Past quizzes generated for a topic, each with its last attempt score if any.
   * @param {string} examId
   * @param {string} topicId
   * @returns {Promise<Object>}
   */
  async listTopicQuizzes(examId, topicId) {
    const response = await apiClient.get(`/exams/${examId}/topics/${topicId}/quizzes`);
    return response.data;
  },

  /**
   * Generates and persists a new quiz for a topic, built from the student's
   * own mapped resources.
   * @param {string} examId
   * @param {string} topicId
   * @param {Object} [options]
   * @param {number} [options.numberOfQuestions]
   * @param {'easy'|'medium'|'hard'} [options.difficulty]
   * @param {string} [options.subtopicId]
   * @returns {Promise<Object>}
   */
  async generateTopicQuiz(examId, topicId, options = {}) {
    const response = await apiClient.post(`/exams/${examId}/topics/${topicId}/quizzes`, options);
    return response.data;
  },

  /**
   * A quiz's questions, without answers — for taking it.
   * @param {string} quizId
   * @returns {Promise<Object>}
   */
  async getQuiz(quizId) {
    const response = await apiClient.get(`/quizzes/${quizId}`);
    return response.data;
  },

  /**
   * Submits answers for grading. Scoring happens server-side, deterministically.
   * @param {string} quizId
   * @param {Array<{questionId: string, selectedAnswer: string}>} answers
   * @returns {Promise<Object>}
   */
  async submitQuizAttempt(quizId, answers) {
    const response = await apiClient.post(`/quizzes/${quizId}/submit`, { answers });
    return response.data;
  },

  /**
   * Past attempts at a quiz.
   * @param {string} quizId
   * @returns {Promise<Object>}
   */
  async listQuizAttempts(quizId) {
    const response = await apiClient.get(`/quizzes/${quizId}/attempts`);
    return response.data;
  },

  /**
   * One past attempt's full per-question breakdown.
   * @param {string} quizId
   * @param {string} attemptId
   * @returns {Promise<Object>}
   */
  async getQuizAttempt(quizId, attemptId) {
    const response = await apiClient.get(`/quizzes/${quizId}/attempts/${attemptId}`);
    return response.data;
  },
};
