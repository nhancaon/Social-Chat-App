import { createRouter, createWebHistory } from 'vue-router'
import store from '@/store'
import RedirectIfAuthenticated from './RedirectIfAuthenticated'
import HomeView from '@/views/HomeView.vue'
import Auth from '@/views/Auth.vue'
import Profile from '@/views/Profile.vue'
import PostDetail from '../components/Post/PostDetail.vue'
import Search from '../components/Search/Search.vue'
import Notification from '../components/Notification/Notification.vue'
import Chat from '../components/Chat/Chat.vue'

const routes = [
  { path: '/', name: 'home', component: HomeView, meta: { requiresAuth: true } },
  {
    path: '/Auth',
    name: 'Auth',
    component: Auth,
    beforeEnter: [RedirectIfAuthenticated],
  },
  { path: '/PostDetail/:id', name: 'PostDetail', component: PostDetail },
  { path: '/Search', name: 'Search', component: Search },
  {
    path: '/Profile/:id',
    name: 'Profile',
    component: Profile,
    meta: { requiresAuth: true },
  },
  {
    path: '/Notification',
    name: 'Notification',
    component: Notification,
    meta: { requiresAuth: true },
  },
  {
    path: '/Chat',
    name: 'chat',
    component: Chat,
    meta: { requiresAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
})

router.beforeEach(async (to, from, next) => {
  if (!store.state.auth.authData) {
    await store.dispatch('auth/initAuth')
  }

  if (to.meta.requiresAuth && !store.getters['auth/isAuthenticated']) {
    return next({ name: 'Auth' })
  }

  if (store.getters['auth/isAuthenticated']) {
    await store.dispatch('users/ensureCurrentUserLoaded')
  }

  next()
})

export default router