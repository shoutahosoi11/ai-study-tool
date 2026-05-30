import { getApps, initializeApp } from 'firebase/app'
import {
  type Auth,
  type User,
  createUserWithEmailAndPassword,
  getAuth,
  inMemoryPersistence,
  initializeAuth,
  onAuthStateChanged,
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

export function onAuthChanged(callback: (user: MobileAuthUser | null) => void) {
  if (!isFirebaseConfigured()) {
    callback(null)
    return () => undefined
  }

  return onAuthStateChanged(getOrCreateAuth(), callback)
}
