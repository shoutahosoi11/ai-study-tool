import { getApps, initializeApp } from 'firebase/app'
import {
  type Auth,
  type User,
  EmailAuthProvider,
  createUserWithEmailAndPassword,
  getAuth,
  inMemoryPersistence,
  initializeAuth,
  onAuthStateChanged,
  reauthenticateWithCredential,
  signInWithEmailAndPassword,
  signOut,
} from 'firebase/auth'

import { firebaseConfig, isFirebaseConfigured } from '../config'

export type MobileAuthUser = User

let authInstance: Auth | null = null

export function getOrCreateFirebaseApp() {
  if (!isFirebaseConfigured()) {
    throw new Error('Firebase configuration is missing in mobile/.env')
  }

  return getApps().length === 0 ? initializeApp(firebaseConfig) : getApps()[0]
}

function getOrCreateAuth(): Auth {
  if (!isFirebaseConfigured()) {
    throw new Error('Firebase configuration is missing in mobile/.env')
  }

  if (authInstance) {
    return authInstance
  }

  const app = getOrCreateFirebaseApp()

  try {
    authInstance = initializeAuth(app, {
      persistence: inMemoryPersistence,
    })
  } catch {
    authInstance = getAuth(app)
  }

  return authInstance
}

export function getCurrentUser(): MobileAuthUser | null {
  if (!isFirebaseConfigured()) {
    return null
  }

  return getOrCreateAuth().currentUser
}

export async function signInWithEmail(email: string, password: string) {
  return signInWithEmailAndPassword(getOrCreateAuth(), email, password)
}

export async function signUpWithEmail(email: string, password: string) {
  return createUserWithEmailAndPassword(getOrCreateAuth(), email, password)
}

export async function signOutUser() {
  return signOut(getOrCreateAuth())
}

export async function getIdToken(): Promise<string | null> {
  const user = getCurrentUser()
  if (!user) {
    return null
  }

  return user.getIdToken()
}

// 退会などの危険操作は auth_time が5分以内であることをサーバーが要求する。
// トークンのリフレッシュでは auth_time は更新されないため、パスワードで
// 再認証してから強制リフレッシュしたトークンを使う。
export async function reauthenticateWithPassword(password: string): Promise<void> {
  const user = getCurrentUser()
  if (!user || !user.email) {
    throw new Error('not signed in')
  }

  await reauthenticateWithCredential(user, EmailAuthProvider.credential(user.email, password))
  await user.getIdToken(true)
}

export function onAuthChanged(callback: (user: MobileAuthUser | null) => void) {
  if (!isFirebaseConfigured()) {
    callback(null)
    return () => undefined
  }

  return onAuthStateChanged(getOrCreateAuth(), callback)
}
