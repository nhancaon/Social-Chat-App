<template>
  <q-header class="app-header" bordered>
    <q-toolbar class="constrain header-toolbar">
      <!-- Logo -->
      <q-btn flat to="/" class="brand-btn gt-xs" no-caps>
        <q-icon size="1.9em" name="eva-camera-outline" class="brand-icon" />
        <q-toolbar-title class="brand-title grand-hotel text-bold">Home</q-toolbar-title>
      </q-btn>

      <q-btn flat to="/" class="brand-btn lt-sm" no-caps>
        <q-icon size="1.5em" name="eva-camera-outline" class="brand-icon" />
        <q-toolbar-title class="brand-title grand-hotel text-bold">Home</q-toolbar-title>
      </q-btn>

      <q-separator class="header-separator gt-xs" vertical spaced />

      <!-- Search (desktop) -->
      <div class="search-wrap gt-xs">
        <q-input v-model="searchText" dense rounded outlined class="search-input" placeholder="Search"
          @keyup.enter="goSearch">
          <template #prepend>
            <q-icon name="eva-search-outline" size="20px" class="search-icon" />
          </template>
        </q-input>
      </div>

      <q-space class="gt-xs" />

      <!-- Right icons (desktop) -->
      <div class="header-actions gt-xs">
        <q-btn round flat unelevated v-show="isLoggedIn" @click="goToChat"
          :icon="getUnreadMsgCount > 0 ? 'eva-message-square-outline' : 'eva-message-square'" class="header-icon-btn"
          :class="{ 'is-active': getUnreadMsgCount > 0 }">
          <q-badge v-if="getUnreadMsgCount > 0" class="nav-badge" floating rounded :label="getUnreadMsgCount" />
          <q-tooltip>Messages</q-tooltip>
        </q-btn>

        <q-btn round flat unelevated v-show="isLoggedIn" @click="goToNotification"
          :icon="getUnreadNotificationCount > 0 ? 'eva-bell-outline' : 'eva-bell'" class="header-icon-btn"
          :class="{ 'is-active': getUnreadNotificationCount > 0 }">
          <q-badge v-if="getUnreadNotificationCount > 0" class="nav-badge" floating rounded
            :label="getUnreadNotificationCount" />
          <q-tooltip>Notifications</q-tooltip>
        </q-btn>

        <q-btn v-show="isLoggedIn" round flat class="avatar-btn">
          <q-avatar size="38px" class="avatar-ring">
            <img v-if="currentUser?.imageUrl" :src="currentUser.imageUrl">
            <img v-else src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png">
          </q-avatar>
          <q-menu anchor="bottom right" self="top right" class="user-menu">
            <q-list style="min-width: 160px" padding>
              <q-item clickable v-close-popup @click="goToProfile">
                <q-item-section avatar><q-icon name="eva-person-outline" class="menu-icon" /></q-item-section>
                <q-item-section>Profile</q-item-section>
              </q-item>
              <q-separator spaced />
              <q-item clickable v-close-popup @click="logUserOut">
                <q-item-section avatar><q-icon name="eva-log-out-outline" class="menu-icon" /></q-item-section>
                <q-item-section>Logout</q-item-section>
              </q-item>
            </q-list>
          </q-menu>
        </q-btn>
      </div>

      <!-- Mobile: avatar + menu btn -->
      <q-space class="lt-sm" />

      <div class="lt-sm mobile-actions">
        <q-btn v-show="isLoggedIn" round flat size="sm">
          <q-avatar size="30px" class="avatar-ring">
            <img v-if="currentUser?.imageUrl" :src="currentUser.imageUrl">
            <img v-else src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png">
          </q-avatar>
          <q-badge v-if="(getUnreadNotificationCount + getUnreadMsgCount) > 0" class="nav-badge" floating rounded
            :label="getUnreadNotificationCount + getUnreadMsgCount" />
        </q-btn>

        <q-btn flat round dense icon="eva-menu-outline" class="header-icon-btn"
          @click="mobileMenuOpen = !mobileMenuOpen" />
      </div>
    </q-toolbar>

    <!-- Mobile drawer -->
    <q-drawer v-model="mobileMenuOpen" side="right" overlay :width="260" bordered class="mobile-drawer">
      <div class="drawer-header">
        <q-icon size="1.6em" name="eva-camera-outline" class="brand-icon" />
        <span class="grand-hotel text-bold">Home</span>
      </div>

      <q-input v-model="searchText" dense outlined rounded class="q-mx-md q-mb-md" placeholder="Search"
        @keyup.enter="() => { goSearch(); mobileMenuOpen = false }">
        <template #prepend><q-icon name="eva-search-outline" class="search-icon" /></template>
      </q-input>

      <q-separator />

      <q-list padding>
        <template v-if="isLoggedIn">
          <q-item clickable v-ripple @click="handleProfileClick">
            <q-item-section avatar><q-icon name="eva-person-outline" class="menu-icon" /></q-item-section>
            <q-item-section>Profile</q-item-section>
          </q-item>

          <q-item clickable v-ripple @click="handleChatClick">
            <q-item-section avatar>
              <div class="mobile-icon-badge">
                <q-icon name="eva-message-square-outline" class="menu-icon" />
                <q-badge v-if="getUnreadMsgCount > 0" class="nav-badge" :label="getUnreadMsgCount" />
              </div>
            </q-item-section>
            <q-item-section>Messages</q-item-section>
          </q-item>

          <q-item clickable v-ripple @click="handleNotificationClick">
            <q-item-section avatar>
              <div class="mobile-icon-badge">
                <q-icon name="eva-bell-outline" class="menu-icon" />
                <q-badge v-if="getUnreadNotificationCount > 0" class="nav-badge" :label="getUnreadNotificationCount" />
              </div>
            </q-item-section>
            <q-item-section>Notifications</q-item-section>
          </q-item>

          <q-separator spaced />

          <q-item clickable v-ripple @click="handleLogoutClick">
            <q-item-section avatar><q-icon name="eva-log-out-outline" class="menu-icon" /></q-item-section>
            <q-item-section>Logout</q-item-section>
          </q-item>
        </template>

        <q-item v-else clickable v-ripple to="/Auth" @click="mobileMenuOpen = false">
          <q-item-section avatar><q-icon name="eva-log-in-outline" class="menu-icon" /></q-item-section>
          <q-item-section>Login</q-item-section>
        </q-item>
      </q-list>
    </q-drawer>
  </q-header>
</template>

<script>
import { mapGetters, mapActions } from 'vuex';

export default {
  name: 'NavBar',
  data() {
    return {
      searchText: '',
      mobileMenuOpen: false // chỉ dùng cho UI drawer, không liên quan logic call API
    };
  },
  computed: {
    ...mapGetters('auth', ['getAuthData', 'currentUserId']),
    ...mapGetters('chat', ['getUnreadMsgCount']),
    ...mapGetters('notification', ['getUnreadNotificationCount']),
    ...mapGetters('realTimeChat', ['getMessageReceivedTick']),

    isLoggedIn() {
      return !!this.getAuthData?.result;
    },
    currentUser() {
      return this.getAuthData?.result;
    }
  },
  watch: {
    getMessageReceivedTick() {
      if (this.currentUserId) {
        this.getUnreadMessageNum(this.currentUserId);
      }
    },
  },
  methods: {
    ...mapActions('auth', ['logout']),
    ...mapActions('realTimeNotify', ['connectToNotifications', 'disconnectFromNotifications']),
    ...mapActions('realTimeChat', ['createChatConnection', 'stopConnectionToChat']),
    ...mapActions('notification', ['getUnreadNotifyNum']),
    ...mapActions('chat', ['getUnreadMessageNum']),
    goSearch() {
      if (!this.searchText) return
      this.$router.push({ path: '/Search', query: { search: this.searchText } })
    },
    goToProfile() {
      const id = this.currentUser?._id
      this.$router.push(`/Profile/${id}`)
    },
    async logUserOut() {
      try {
        await this.stopConnectionToChat();
        await this.disconnectFromNotifications();
      } finally {
        await this.logout();
        this.$router.push('/Auth');
      }
    },
    goToNotification() {
      this.$router.push('/Notification')
    },
    goToChat() {
      this.$router.push('/Chat')
    },
    handleVisibilityChange() {
      if (document.visibilityState === 'visible' && this.isLoggedIn) {
        this.connectToNotifications();
      }
    },
    handleProfileClick() {
      this.mobileMenuOpen = false;
      this.goToProfile();
    },
    handleChatClick() {
      this.mobileMenuOpen = false;
      this.goToChat();
    },
    handleNotificationClick() {
      this.mobileMenuOpen = false;
      this.goToNotification();
    },
    handleLogoutClick() {
      this.mobileMenuOpen = false;
      this.logUserOut();
    }
  },
  async mounted() {
    if (!this.isLoggedIn) return;
    const id = this.currentUserId;
    if (id) {
      try {
        await this.getUnreadNotifyNum(id);
        await this.getUnreadMessageNum(id);
      } catch (error) {
        console.log('Lỗi lấy số liệu ban đầu:', error);
      }
      await this.connectToNotifications();
      await this.createChatConnection();
      document.addEventListener('visibilitychange', this.handleVisibilityChange);
    }
  },
  beforeUnmount() {
    document.removeEventListener('visibilitychange', this.handleVisibilityChange);
  },
};
</script>

<style lang="sass">
.app-header
  --nav-bg: #FFFFFF
  --nav-ink: #0A0A0A
  --nav-ink-soft: #737373
  --nav-accent: #000000
  --nav-accent-soft: #F5F5F5
  --nav-danger: #DC2626
  --nav-border: #E5E5E5
  background: var(--nav-bg) !important

.header-toolbar
  padding: 8px 16px
  min-height: 64px

// ---- Brand ----
.brand-btn
  padding: 4px 8px
  .brand-icon
    color: var(--nav-ink)
  .brand-title
    font-size: 1.6em
    margin-left: 6px
    letter-spacing: .5px
    color: var(--nav-ink)

.header-separator
  margin: 0 16px
  height: 32px
  background: var(--nav-border)

// ---- Search ----
.search-wrap
  flex: 1
  max-width: 380px

.search-input
  width: 100%
  .q-field__control
    background: #F9FAFB
  .search-icon
    color: var(--nav-ink-soft)

// ---- Icon buttons (chat / notification / menu) ----
.header-icon-btn
  margin-left: 4px
  color: var(--nav-ink) !important
  background: transparent
  transition: background-color .15s ease, color .15s ease

  &:hover
    background: var(--nav-accent-soft)

  &.is-active
    color: var(--nav-accent) !important

.nav-badge
  background: var(--nav-danger) !important
  color: #fff !important
  min-width: 18px
  height: 18px
  font-size: 11px
  font-weight: 600

// ---- Avatar ----
.avatar-btn
  margin-left: 8px

.avatar-ring
  border: 2px solid var(--nav-border)
  transition: border-color .2s ease
  &:hover
    border-color: var(--nav-accent)

// ---- Dropdown menu ----
.user-menu
  .menu-icon
    color: var(--nav-ink-soft)

.menu-icon
  color: var(--nav-ink-soft)

// ---- Mobile ----
.mobile-actions
  display: flex
  align-items: center
  gap: 4px

.drawer-header
  display: flex
  align-items: center
  gap: 8px
  padding: 16px
  font-size: 1.3em
  color: var(--nav-ink)
  .brand-icon
    color: var(--nav-ink)

.mobile-icon-badge
  position: relative
  display: inline-block
  .q-badge
    position: absolute
    top: -6px
    right: -6px

@media (max-width: 599px)
  .header-toolbar
    padding: 6px 12px
    min-height: 56px
</style>