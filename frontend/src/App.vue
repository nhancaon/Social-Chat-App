<template>
  <q-layout view="lHh Lpr lFf">
    <NavBar />
    <q-page-container class="bg-grey-1">
      <router-view />

    </q-page-container>
  </q-layout>
</template>

<script>
import NavBar from '@/views/NavBar.vue'
import { mapActions, mapGetters } from 'vuex'

export default {
  name: 'MainLayout',
  methods: {
    ...mapActions('auth', ['initAuth']),
    ...mapActions('realTimeNotify', ['connectToNotifications', "disconnectFromNotifications"]),
    ...mapActions('realTimeChat', ['createChatConnection', 'stopConnectionToChat']),
    handleVisibilityChange() {
      if (!document.hidden) {
        // Tab đang được hiển thị
        if (this.isAuthenticated && !this.isChatConnected) {
          console.log('🔄 Reconnecting WebSocket after tab focus');
          this.createChatConnection();
          this.connectToNotifications();
        }
      }
    },
  },
  computed: {
    ...mapGetters('auth', ['isAuthenticated']),
  },
  mounted() {
    document.addEventListener('visibilitychange', this.handleVisibilityChange);
  },
  beforeUnmount() {
    document.removeEventListener('visibilitychange', this.handleVisibilityChange);
    if (this.reconnectInterval) clearInterval(this.reconnectInterval);
  },
  components: { NavBar },
}
</script>

<style lang="scss">
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}

nav {
  padding: 30px;

  a {
    font-weight: bold;
    color: #2c3e50;

    &.router-link-exact-active {
      color: #42b983;
    }
  }
}
</style>