import { createRouter, createWebHistory } from 'vue-router'
import RedirectIfAuthenticated from "./RedirectIfAuthenticated"
import HomeView from '@/views/HomeView.vue'
import Auth from '@/views/Auth.vue'
// import Profile from '@/views/Profile.vue'
// import PostDeatils from '../components/post/PostDeatils.vue';
// import Search from '../components/search/Search.vue';
// import Notification from '../components/Notification/Notification.vue';
// import Chat from '../components/Chat/Chat.vue'

const routes = [
  {
    path: '/',
    name: 'home',
    component: HomeView
  },
  {
    path: '/Auth',
    name: 'Auth',
    component: Auth,
    beforeEnter: [RedirectIfAuthenticated]
  },
  // {
  //   path: '/PostDeatils/:id',
  //   name: "PostDeatils",
  //   component: PostDeatils,
  // },
  // {
  //   path: '/Search',
  //   name: 'Search',
  //   component: Search,
  // },
  // {
  //   path: '/Profile/:id',
  //   name: 'Profile',
  //   component: Profile
  // },
  // {
  //   path: '/Notification',
  //   name: 'Notification',
  //   component: Notification
  // },
  // {
  //   path: '/Chat',
  //   name: 'chat',
  //   component: Chat,
  // }
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes
})

export default router