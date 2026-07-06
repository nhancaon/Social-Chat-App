import store from '@/store'

export default function RedirectIfAuthenticated(to, from, next) {
  if (store.getters['auth/isAuthenticated']) {
    next('/')
  } else {
    next()
  }
}