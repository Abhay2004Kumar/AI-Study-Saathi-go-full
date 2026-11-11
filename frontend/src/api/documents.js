import apiClient from './client';

export const documentService = {
  /**
   * Upload a new document
   * @param {Object} file - The file object from expo-document-picker
   * @param {string} title - The title of the document
   * @param {Object} [options]
   * @param {string} [options.examId] - Link this resource to an exam's syllabus
   * @param {'SYLLABUS'|'NOTES'|'PYQ'|'GENERAL'} [options.category]
   * @returns {Promise<Object>}
   */
  async uploadDocument(file, title, options = {}) {
    const formData = new FormData();

    // React Native requires a specific format for file uploads in FormData
    formData.append('title', title);
    if (options.examId) formData.append('examId', options.examId);
    if (options.category) formData.append('category', options.category);
    formData.append('file', {
      uri: file.uri,
      name: file.name,
      type: file.mimeType || 'application/octet-stream',
    });

    const response = await apiClient.post('/documents', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });

    return response.data;
  },

  /**
   * Get all user documents, optionally scoped to one exam.
   * @param {Object} [options]
   * @param {string} [options.examId]
   * @returns {Promise<Object>}
   */
  async getDocuments(options = {}) {
    const response = await apiClient.get('/documents', {
      params: options.examId ? { examId: options.examId } : undefined,
    });
    return response.data;
  },

  /**
   * Check processing status of a document
   * @param {string} id - Document ID
   * @returns {Promise<Object>}
   */
  async getDocumentStatus(id) {
    const response = await apiClient.get(`/documents/${id}/status`);
    return response.data;
  },

  /**
   * Get a single document
   * @param {string} id
   * @returns {Promise<Object>}
   */
  async getDocument(id) {
    const response = await apiClient.get(`/documents/${id}`);
    return response.data;
  },

  /**
   * Update a document's category — e.g. to accept the AI classifier's suggestion.
   * @param {string} id
   * @param {'SYLLABUS'|'NOTES'|'PYQ'|'GENERAL'} category
   * @returns {Promise<Object>}
   */
  async updateDocumentCategory(id, category) {
    const response = await apiClient.patch(`/documents/${id}`, { category });
    return response.data;
  },

  /**
   * Delete a document
   * @param {string} id
   * @returns {Promise<Object>}
   */
  async deleteDocument(id) {
    const response = await apiClient.delete(`/documents/${id}`);
    return response.data;
  },

  /**
   * Retry processing for a document stuck in FAILED status.
   * @param {string} id
   * @returns {Promise<Object>}
   */
  async retryDocument(id) {
    const response = await apiClient.post(`/documents/${id}/retry`);
    return response.data;
  },
};
