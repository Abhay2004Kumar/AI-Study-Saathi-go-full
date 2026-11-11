import apiClient from './client';
import * as SecureStore from 'expo-secure-store';

export const authService = {
  async register(name, email, password) {
    const response = await apiClient.post('/auth/register', { name, email, password });
    if (response.data?.data?.token) {
      await SecureStore.setItemAsync('token', response.data.data.token);
      await SecureStore.setItemAsync('user', JSON.stringify(response.data.data));
    }
    return response.data;
  },

  async login(email, password) {
    const response = await apiClient.post('/auth/login', { email, password });
    if (response.data?.data?.token) {
      await SecureStore.setItemAsync('token', response.data.data.token);
      await SecureStore.setItemAsync('user', JSON.stringify(response.data.data));
    }
    return response.data;
  },

  async logout() {
    await SecureStore.deleteItemAsync('token');
    await SecureStore.deleteItemAsync('user');
  },

  async getToken() {
    return await SecureStore.getItemAsync('token');
  },
  
  async getUser() {
    const user = await SecureStore.getItemAsync('user');
    return user ? JSON.parse(user) : null;
  }
};
