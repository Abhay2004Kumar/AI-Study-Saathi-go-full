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

export default function RegisterScreen() {
  const { register } = useAuth();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const keyboardVisible = useKeyboardVisible();
  const emailRef = useRef<TextInput>(null);
  const passwordRef = useRef<TextInput>(null);

  const handleRegister = async () => {
    if (!name || !email || !password) {
      Alert.alert('Missing Details', 'Please fill in all fields');
      return;
    }
    try {
      setLoading(true);
      await register(name.trim(), email.trim(), password.trim());
    } catch (error: any) {
      Alert.alert('Registration Failed', error.response?.data?.message || error.message);
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
                <Text style={styles.emoji}>🎓</Text>
              </View>
              <View style={styles.emojiCircleSmall1}>
                <Text style={styles.emojiSmall}>✏️</Text>
              </View>
              <View style={styles.emojiCircleSmall2}>
                <Text style={styles.emojiSmall}>🏆</Text>
              </View>
            </>
          )}
        </View>

        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.title}>Join AI Study Saathi 🌟</Text>
          <Text style={styles.subtitle}>Create your free account and start learning smarter</Text>
        </View>

        {/* Form card */}
        <View style={styles.formCard}>
          <AppTextInput
            label="Your Name"
            placeholder="E.g. Riya Sharma"
            autoCapitalize="words"
            autoComplete="name"
            textContentType="name"
            value={name}
            onChangeText={setName}
            icon="👤"
            returnKeyType="next"
            submitBehavior="submit"
            onSubmitEditing={() => emailRef.current?.focus()}
          />
          <AppTextInput
            ref={emailRef}
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
            placeholder="Create a strong password"
            secureTextEntry
            autoCapitalize="none"
            autoComplete="new-password"
            textContentType="newPassword"
            value={password}
            onChangeText={setPassword}
            icon="🔒"
            returnKeyType="go"
            onSubmitEditing={handleRegister}
          />

          <AppButton
            title="Create Account"
            onPress={handleRegister}
            loading={loading}
            style={styles.registerBtn}
          />

          <View style={styles.footer}>
            <Text style={styles.footerText}>Already have an account? </Text>
            <Link href={"/(auth)/login" as any} asChild>
              <TouchableOpacity>
                <Text style={styles.linkText}>Log In →</Text>
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
    height: 140,
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
    width: 100,
    height: 100,
    borderRadius: 50,
    backgroundColor: colors.successLight,
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 3,
    borderColor: colors.success + '40',
  },
  emoji: {
    fontSize: 48,
  },
  emojiCircleSmall1: {
    position: 'absolute',
    top: 5,
    right: 55,
    width: 38,
    height: 38,
    borderRadius: 19,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center',
    alignItems: 'center',
  },
  emojiCircleSmall2: {
    position: 'absolute',
    bottom: 5,
    left: 55,
    width: 38,
    height: 38,
    borderRadius: 19,
    backgroundColor: colors.surfaceWarm,
    borderWidth: 1,
    borderColor: colors.border,
    justifyContent: 'center',
    alignItems: 'center',
  },
  emojiSmall: {
    fontSize: 18,
  },
  header: {
    marginBottom: 24,
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
  registerBtn: {
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
