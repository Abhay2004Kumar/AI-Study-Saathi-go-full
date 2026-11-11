import apiClient from './client';

export const studyPlanService = {
  /**
   * The exam's currently active study plan, or null if none has been
   * generated yet.
   * @param {string} examId
   * @returns {Promise<Object>}
   */
  async getPlan(examId) {
    const response = await apiClient.get(`/exams/${examId}/study-plan`);
    return response.data;
  },

  /**
   * (Re)generates the exam's study plan from its current syllabus, mastery,
   * and PYQ importance — replaces any existing plan outright.
   * @param {string} examId
   * @returns {Promise<Object>}
   */
  async generatePlan(examId) {
    const response = await apiClient.post(`/exams/${examId}/study-plan`);
    return response.data;
  },

  /**
   * Marks one session done or not-done.
   * @param {string} sessionId
   * @param {boolean} completed
   * @returns {Promise<Object>}
   */
  async setSessionCompletion(sessionId, completed) {
    const response = await apiClient.patch(`/study-plan-sessions/${sessionId}`, { completed });
    return response.data;
  },
};
