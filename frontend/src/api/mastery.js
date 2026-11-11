import apiClient from './client';

export const masteryService = {
  /**
   * This user's deterministic mastery score for one topic, computed live
   * from their quiz attempt history (recency-weighted accuracy).
   * @param {string} examId
   * @param {string} topicId
   * @returns {Promise<Object>}
   */
  async getTopicMastery(examId, topicId) {
    const response = await apiClient.get(`/exams/${examId}/topics/${topicId}/mastery`);
    return response.data;
  },

  /**
   * Every topic under this exam with its mastery score, weakest first —
   * the "weakness" view for deciding where to study next.
   * @param {string} examId
   * @returns {Promise<Object>}
   */
  async getExamMastery(examId) {
    const response = await apiClient.get(`/exams/${examId}/mastery`);
    return response.data;
  },
};
