import { useState } from 'react'
import { Pressable, Text, TextInput, View } from 'react-native'

import { styles } from '../appStyles'

export type FieldProps = {
  label: string
  value: string
  placeholder?: string
  autoCapitalize?: 'none' | 'sentences' | 'words' | 'characters'
  keyboardType?: 'default' | 'email-address' | 'url'
  secureTextEntry?: boolean
  multiline?: boolean
  minHeight?: number
  autoGrow?: boolean
  onChangeText: (value: string) => void
}

export function Field({
  label,
  value,
  placeholder,
  autoCapitalize = 'sentences',
  keyboardType = 'default',
  secureTextEntry = false,
  multiline = false,
  minHeight,
  autoGrow = false,
  onChangeText,
}: FieldProps) {
  const baseHeight = minHeight ?? (multiline ? 120 : 48)
  const [contentHeight, setContentHeight] = useState(baseHeight)

  return (
    <View style={styles.fieldGroup}>
      <Text style={styles.fieldLabel}>{label}</Text>
      <TextInput
        style={[
          styles.input,
          multiline ? styles.multilineInput : null,
          { minHeight: baseHeight },
          autoGrow && multiline ? { height: Math.max(baseHeight, contentHeight) } : null,
        ]}
        value={value}
        placeholder={placeholder}
        placeholderTextColor="#8a98a8"
        autoCapitalize={autoCapitalize}
        keyboardType={keyboardType}
        secureTextEntry={secureTextEntry}
        multiline={multiline}
        textAlignVertical={multiline ? 'top' : 'center'}
        onContentSizeChange={
          autoGrow && multiline
            ? (event) => {
                setContentHeight(event.nativeEvent.contentSize.height)
              }
            : undefined
        }
        onChangeText={onChangeText}
      />
    </View>
  )
}

export function PrimaryButton({
  label,
  disabled,
  onPress,
}: {
  label: string
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.primaryButton,
        disabled ? styles.primaryButtonDisabled : null,
        pressed && !disabled ? styles.primaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={styles.primaryButtonText}>{label}</Text>
    </Pressable>
  )
}

export function SecondaryButton({
  label,
  disabled,
  onPress,
}: {
  label: string
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.secondaryButton,
        disabled ? styles.secondaryButtonDisabled : null,
        pressed && !disabled ? styles.secondaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={[styles.secondaryButtonText, disabled ? styles.secondaryButtonTextDisabled : null]}>{label}</Text>
    </Pressable>
  )
}

export function SegmentButton({
  label,
  active,
  onPress,
}: {
  label: string
  active: boolean
  onPress: () => void
}) {
  return (
    <Pressable accessibilityRole="button" style={[styles.segmentButton, active ? styles.segmentButtonActive : null]} onPress={onPress}>
      <Text style={[styles.segmentButtonText, active ? styles.segmentButtonTextActive : null]}>{label}</Text>
    </Pressable>
  )
}

export function TabButton({
  icon,
  label,
  active,
  onPress,
}: {
  icon: string
  label: string
  active: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.tabButton,
        active ? styles.tabButtonActive : null,
        pressed ? styles.tabButtonPressed : null,
      ]}
      accessibilityLabel={label}
      onPress={onPress}
    >
      <Text style={[styles.tabButtonText, active ? styles.tabButtonTextActive : null]}>{icon}</Text>
    </Pressable>
  )
}

export function SmallButton({
  label,
  disabled,
  onPress,
}: {
  label: string
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.smallButton,
        disabled ? styles.secondaryButtonDisabled : null,
        pressed && !disabled ? styles.secondaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={styles.smallButtonText}>{label}</Text>
    </Pressable>
  )
}

export function IconButton({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" style={styles.iconButton} onPress={onPress}>
      <Text style={styles.iconButtonText}>{label}</Text>
    </Pressable>
  )
}

export function AvatarChip({ label }: { label: string }) {
  return (
    <View style={styles.avatarChip}>
      <Text style={styles.avatarChipText}>{label}</Text>
    </View>
  )
}

export function ProfileStat({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.profileStat}>
      <Text style={styles.profileStatValue}>{value}</Text>
      <Text style={styles.profileStatLabel}>{label}</Text>
    </View>
  )
}
