import React from 'react';
import { Stack } from 'expo-router';
import { colors } from '../../theme/colors';
import { typography } from '../../theme/typography';

export default function AppLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        headerStyle: {
          backgroundColor: colors.surface,
        },
        headerTintColor: colors.text,
        headerShadowVisible: false,
        headerTitleStyle: {
          ...typography.subtitle,
          color: colors.text,
        },
      }}
    >
      <Stack.Screen 
        name="index" 
        options={{ 
          title: 'Study Saathi 📚',
          headerShown: false,
        }} 
      />
    </Stack>
  );
}
