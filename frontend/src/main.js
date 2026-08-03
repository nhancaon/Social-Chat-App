import { createApp } from 'vue'
import App from './App.vue'
import { Quasar } from 'quasar'
import quasarUserOptions from './quasar-user-options'
import router from './router'
import store from './store'

// Fix: ResizeObserver loop completed with undelivered notifications
const debounce = (callback, delay) => {
  let tid
  return function (...args) {
    clearTimeout(tid)
    tid = setTimeout(() => callback.apply(this, args), delay)
  }
}

const _ResizeObserver = window.ResizeObserver
window.ResizeObserver = class ResizeObserver extends _ResizeObserver {
  constructor(callback) {
    super(debounce(callback, 20))
  }
}

createApp(App).use(store).use(router).use(Quasar, quasarUserOptions).mount('#app')