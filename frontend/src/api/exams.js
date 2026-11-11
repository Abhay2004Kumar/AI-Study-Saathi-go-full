import apiClient from './client';

export const examService = {
  /**
   * Create an exam, optionally with a nested syllabus tree.
   * @param {Object} payload - { name, description?, examDate?, availableHoursPerDay?, subjects? }
   * @returns {Promise<Object>}
   */
  async createExam(payload) {
    const response = await apiClient.post('/exams', payload);
    return response.data;
  },

  /**
   * List the current user's exams.
   * @returns {Promise<Object>}
   */
  async getExams() {
    const response = await apiClient.get('/exams');
    return response.data;
  },

  /**
   * Get one exam with its full nested syllabus tree.
   * @param {string} id
   * @returns {Promise<Object>}
   */
  async getExam(id) {
    const response = await apiClient.get(`/exams/${id}`);
    return response.data;
  },

  /**
   * Update exam metadata (name/description/examDate/availableHoursPerDay).
   * @param {string} id
   * @param {Object} payload
   * @returns {Promise<Object>}
   */
  async updateExam(id, payload) {
    const response = await apiClient.patch(`/exams/${id}`, payload);
    return response.data;
  },

  /**
   * Delete an exam and its whole syllabus tree.
   * @param {string} id
   * @returns {Promise<Object>}
   */
  async deleteExam(id) {
    const response = await apiClient.delete(`/exams/${id}`);
    return response.data;
  },

  /**
   * Replace an exam's syllabus tree. Nodes with a matching id/name are kept
   * (and their id preserved); anything missing from `subjects` is deleted.
   * @param {string} id
   * @param {Array} subjects
   * @returns {Promise<Object>}
   */
  async replaceSyllabus(id, subjects) {
    const response = await apiClient.put(`/exams/${id}/syllabus`, { subjects });
    return response.data;
  },

  /**
   * AI-assisted syllabus extraction from pasted text. Returns a suggested
   * { subjects } tree that is NOT persisted — pass it to createExam or
   * replaceSyllabus (after the user reviews/edits it) to actually save it.
   * @param {string} text
   * @returns {Promise<Object>}
   */
  async extractSyllabus(text) {
    const response = await apiClient.post('/exams/syllabus/extract', { text });
    return response.data;
  },
};
