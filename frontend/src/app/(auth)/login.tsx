import React, { useRef, useState } from 'react';
import { View, Text, StyleSheet, Alert, TouchableOpacity, TextInput } from 'react-native';
import { Link } from 'expo-router';
import { useAuth } from '../../context/AuthContext';
import AppTextInput from '../../components/AppTextInput';
import AppButton from '../../components/AppButton';
import KeyboardAwareScrollView from '../../components/KeyboardAwareScrollView';
import { useKeyboardVisible } from '../../hooks/use-keyboard';
import { colors } from '../../theme/colors';
import { typography } from '../../theme/typography';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function LoginScreen() {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const keyboardVisible = useKeyboardVisible();
  const passwordRef = useRef<TextInput>(null);

  const handleLogin = async () => {
    if (!email || !password) {
      Alert.alert('Missing Details', 'Please enter both email and password');
      return;
    }
    try {
      setLoading(true);
      await login(email.trim(), password.trim());
    } catch (error: any) {
      Alert.alert('Login Failed', error.response?.data?.message || error.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView style={styles.safeArea}>
      <KeyboardAwareScrollView contentContainerStyle={styles.scrollContent}>
        {/* Illustration area — collapses while typing to free up screen space */}
        <View style={[styles.illustrationArea, keyboardVisible && styles.illustrationAreaHidden]}>
          {!keyboardVisible && (
            <>
              <View style={styles.emojiCircle}>
                <Text style={styles.emoji}>📚</Text>
              </View>
              <View style={styles.emojiCircleSmall1}>
                <Text style={styles.emojiSmall}>⭐</Text>
              </View>
              <View style={styles.emojiCircleSmall2}>
                <Text style={styles.emojiSmall}>🎯</Text>
              </View>
            </>
          )}
        </View>

        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.title}>Welcome Back! 👋</Text>
          <Text style={styles.subtitle}>Ready to continue your learning journey?</Text>
        </View>

        {/* Form card */}
        <View style={styles.formCard}>
          <AppTextInput
            label="Email Address"
            placeholder="you@example.com"
            keyboardType="email-address"
            autoCapitalize="none"
            autoCorrect={false}
            autoComplete="email"
            textContentType="emailAddress"
            value={email}
            onChangeText={setEmail}
            icon="✉️"
            returnKeyType="next"
            submitBehavior="submit"
            onSubmitEditing={() => passwordRef.current?.focus()}
          />
          <AppTextInput
            ref={passwordRef}
            label="Password"
            placeholder="Enter your password"
            secureTextEntry
            autoCapitalize="none"
            autoComplete="current-password"
            textContentType="password"
            value={password}
            onChangeText={setPassword}
            icon="🔒"
            returnKeyType="go"
            onSubmitEditing={handleLogin}
          />

          <AppButton
            title="Log In"
            onPress={handleLogin}
            loading={loading}
            style={styles.loginBtn}
          />

          <View style={styles.footer}>
            <Text style={styles.footerText}>New here? </Text>
            <Link href={"/(auth)/register" as any} asChild>
              <TouchableOpacity>
                <Text style={styles.linkText}>Create an Account →</Text>
              </TouchableOpacity>
            </Link>
          </View>
        </View>
      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  scrollContent: {
    flexGrow: 1,
    paddingHorizontal: 24,
    paddingTop: 20,
    paddingBottom: 40,
    // Keeps the form a readable width on tablets and in landscape.
    width: '100%',
    maxWidth: 520,
    alignSelf: 'center',
  },
  illustrationArea: {
    height: 160,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 20,
    position: 'relative',
  },
  illustrationAreaHidden: {
    height: 0,
    marginBottom: 0,
  },
  emojiCircle: {
    width: 110,
    height: 110,
    borderRadius: 55,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 3,
    borderColor: colors.primary + '40',
  },
  emoji: {
    fontSize: 52,
  },
  emojiCircleSmall1: {
    position: 'absolute',
    top: 10,
    right: 50,
    width: 42,
    height: 42,
    borderRadius: 21,
    backgroundColor: colors.successLight,
    justifyContent: 'center',
    alignItems: 'center',
  },
  emojiCircleSmall2: {
    position: 'absolute',
    bottom: 10,
    left: 50,
    width: 42,
    height: 42,
    borderRadius: 21,
    backgroundColor: colors.surfaceWarm,
    borderWidth: 1,
    borderColor: colors.border,
    justifyContent: 'center',
    alignItems: 'center',
  },
  emojiSmall: {
    fontSize: 20,
  },
  header: {
    marginBottom: 28,
  },
  title: {
    ...typography.header,
    color: colors.text,
    marginBottom: 6,
  },
  subtitle: {
    ...typography.body,
    color: colors.textMuted,
  },
  formCard: {
    backgroundColor: colors.surface,
    borderRadius: 24,
    padding: 24,
    borderWidth: 1,
    borderColor: colors.border,
    shadowColor: colors.text,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.06,
    shadowRadius: 12,
    elevation: 3,
  },
  loginBtn: {
    marginTop: 8,
    marginBottom: 4,
    marginVertical: 0,
  },
  footer: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    marginTop: 20,
  },
  footerText: {
    ...typography.body,
    color: colors.textMuted,
  },
  linkText: {
    ...typography.body,
    color: colors.primary,
    fontWeight: '700',
  },
});
