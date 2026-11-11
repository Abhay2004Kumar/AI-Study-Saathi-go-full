import React, { createContext, useContext, useEffect, useState } from 'react';
import { authService } from '../api/auth';
import { useRouter, useSegments } from 'expo-router';

const AuthContext = createContext<any>(null);

export function useAuth() {
  return useContext(AuthContext);
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const segments = useSegments();
  const router = useRouter();

  useEffect(() => {
    // Check if user is logged in on mount
    authService.getUser().then((storedUser) => {
      if (storedUser) {
        setUser(storedUser);
      }
      setIsLoading(false);
    });
  }, []);

  useEffect(() => {
    if (isLoading) return;

    const inAuthGroup = (segments[0] as string) === '(auth)';

    if (!user && !inAuthGroup) {
      // Redirect to login if not logged in
      router.replace('/(auth)/login' as any);
    } else if (user && inAuthGroup) {
      // Redirect to dashboard if logged in
      router.replace('/(app)' as any);
    }
  }, [user, segments, isLoading]);

  const login = async (email: string, password: string) => {
    const data = await authService.login(email, password);
    setUser(data.data);
  };

  const register = async (name: string, email: string, password: string) => {
    const data = await authService.register(name, email, password);
    setUser(data.data);
  };

  const logout = async () => {
    await authService.logout();
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, isLoading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
