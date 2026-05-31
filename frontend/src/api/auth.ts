import { initializeApp, getApps } from 'firebase/app'
import axios from 'axios'
import {
  getAuth,
  signInWithEmailAndPassword,
  createUserWithEmailAndPassword,
  signOut,
  onAuthStateChanged,
  GoogleAuthProvider,
  signInWithPopup,
  type User,
} from 'firebase/auth'

const apiBaseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1'
const csrfCookieName = 'csrf_token'

const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
}

const app = getApps().length === 0 ? initializeApp(firebaseConfig) : getApps()[0]
export const auth = getAuth(app)

let authReadyPromise: Promise<void> | null = null
let webSessionPromise: Promise<boolean> | null = null
let webSessionUID = ''
let webSessionSyncStarted = false

function waitForAuthReady(): Promise<void> {
  if (authReadyPromise) return authReadyPromise

  authReadyPromise = new Promise(function (resolve) {
    const unsubscribe = onAuthStateChanged(auth, function () {
      unsubscribe()
      resolve()
    })
  })
  return authReadyPromise
}

export async function signInWithEmail(email: string, password: string) {
  return signInWithEmailAndPassword(auth, email, password)
}

export async function signUpWithEmail(email: string, password: string) {
  return createUserWithEmailAndPassword(auth, email, password)
}

export async function signInWithGoogle() {
  const provider = new GoogleAuthProvider()
  return signInWithPopup(auth, provider)
}

export async function signOutUser() {
  let csrfToken = getStoredCSRFToken()
  if (!csrfToken && auth.currentUser) {
    try {
      if (await createWebSession(true)) {
        csrfToken = getStoredCSRFToken()
      }
    } catch (error) {
      console.warn('Web session refresh before logout failed', {
        status: getErrorStatus(error),
      })
    }
  }
  if (csrfToken) {
    try {
      await axios.post(
        '/auth/logout',
        {},
        {
          baseURL: apiBaseURL,
          withCredentials: true,
          headers: {
            'X-CSRF-Token': csrfToken,
          },
        }
      )
    } catch (error) {
      console.warn('Server logout failed; continuing local sign out', {
        status: getErrorStatus(error),
      })
    }
  }
  resetWebSessionState()
  return signOut(auth)
}

export async function getIdToken(forceRefresh = false): Promise<string | null> {
  await waitForAuthReady()
  const user = auth.currentUser
  if (!user) return null
  return user.getIdToken(forceRefresh)
}

export function onAuthChanged(callback: (user: User | null) => void) {
  startWebSessionSync()
  return onAuthStateChanged(auth, callback)
}

export function getStoredCSRFToken() {
  if (typeof document === 'undefined') {
    return ''
  }

  const cookie = document.cookie
    .split(';')
    .map(function (entry) {
      return entry.trim()
    })
    .find(function (entry) {
      return entry.startsWith(`${csrfCookieName}=`)
    })
  if (!cookie) return ''

  const token = cookie.slice(csrfCookieName.length + 1)
  try {
    return decodeURIComponent(token)
  } catch {
    return token
  }
}

export async function createWebSession(forceRefresh = false): Promise<boolean> {
  await waitForAuthReady()
  const user = auth.currentUser
  if (!user) {
    return false
  }

  const idToken = await user.getIdToken(forceRefresh)
  await axios.post(
    '/auth/session',
    { id_token: idToken },
    {
      baseURL: apiBaseURL,
      withCredentials: true,
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )
  return true
}

export async function ensureWebSession(forceRefresh = false) {
  await waitForAuthReady()
  const user = auth.currentUser
  if (!user) {
    resetWebSessionState()
    return
  }

  if (!forceRefresh && webSessionPromise && webSessionUID === user.uid) {
    return webSessionPromise
  }

  webSessionUID = user.uid
  webSessionPromise = createWebSession(forceRefresh).catch(function (error) {
    if (webSessionUID === user.uid) {
      webSessionPromise = null
    }
    throw error
  })
  return webSessionPromise
}

function resetWebSessionState() {
  webSessionPromise = null
  webSessionUID = ''
}

function startWebSessionSync() {
  if (webSessionSyncStarted) return
  webSessionSyncStarted = true

  onAuthStateChanged(auth, function (user) {
    if (!user) {
      resetWebSessionState()
      return
    }

    ensureWebSession().catch(function (error) {
      console.warn('Web session refresh failed', { status: getErrorStatus(error) })
    })
  })
}

function getErrorStatus(error: unknown) {
  return (error as { response?: { status?: number } }).response?.status ?? null
}
