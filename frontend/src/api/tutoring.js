import apiClient from './client';

export const tutoringService = {
  /**
   * Start a new tutoring session
   * @param {string} subject - Subject to study
   * @param {string} topic - Specific topic to study
   * @returns {Promise<Object>}
   */
  async startSession(subject, topic) {
    const response = await apiClient.post('/tutoring/session/start', { subject, topic });
    return response.data;
  },

  /**
   * Respond to the tutor
   * @param {string} sessionId - The ID of the current tutoring session
   * @param {string} answer - The user's answer
   * @returns {Promise<Object>}
   */
  async respondToTutor(sessionId, answer) {
    const response = await apiClient.post(`/tutoring/session/${sessionId}/respond`, { answer });
    return response.data;
  },

  /**
   * Get an existing session (useful to restore chat history)
   * @param {string} sessionId - The session ID
   * @returns {Promise<Object>}
   */
  async getSession(sessionId) {
    const response = await apiClient.get(`/tutoring/session/${sessionId}`);
    return response.data;
  }
};
