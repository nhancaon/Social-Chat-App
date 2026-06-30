<template>
 <q-header class="bg-white text-grey-10" bordered>
  <q-toolbar class="constrain x">
    <q-btn flat to="/">
        <q-icon left size="3em" name="eva-camera-outline" />
        <q-toolbar-title class="text grand-hotel text-bold"> Home</q-toolbar-title>
    </q-btn>

    <q-separator class="large-screen-only" vertical spaced />

    <q-toolbar-title class="text-center">
        <q-input
          v-model="searchText"
          bottom-slots
          class="nuks"
          label="search"
          @keyup.enter="GoSearch"
        />
    </q-toolbar-title>

    <q-btn round
      v-show="isLoggedIn"
      @click="GoToChat"
      :icon="unReadedMessages > 0 ? 'eva-message-square-outline' : 'eva-message-square'"
      :color="unReadedMessages > 0 ? 'primary' : 'dark'"
    >
     <q-badge v-if="unReadedMessages > 0" color="negative" floating rounded :label="unReadedMessages" />
    </q-btn>

    <q-btn round
      v-show="isLoggedIn"
      @click="GoToNotification"
      :icon="notificationNum > 0 ? 'eva-bell-outline' : 'eva-bell'"
      :color="notificationNum > 0 ? 'primary' : 'dark'"
    >
     <q-badge v-if="notificationNum > 0" floating color="negative" rounded :label="notificationNum" />
    </q-btn>

    <q-btn v-show="isLoggedIn" round>
     <q-avatar size="42px" v-if="currentUser?.imageUrl">
        <img :src="currentUser.imageUrl">
     </q-avatar>
     <q-avatar size="42px" v-else>
        <img src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png">
     </q-avatar>
     <q-menu>
        <q-list style="min-width: 100px">
            <q-item clickable v-close-popup @click="Profile">
                <q-item-section>Profile</q-item-section>
            </q-item>
            <q-separator />
            <q-item clickable v-close-popup @click="LogUserOut">
                <q-item-section>Logout</q-item-section>
            </q-item>
        </q-list>
     </q-menu>
    </q-btn>

  </q-toolbar>
 </q-header>
</template>

<script>
import { mapGetters, mapActions } from 'vuex'

export default {
  name: 'NavBar',
  data() {
    return {
      notificationNum: 0,
      unReadedMessages: 0,
      searchText: '',
    }
  },
  computed: {
    ...mapGetters(['GetAuthData']),
    isLoggedIn() {
      return !!this.GetAuthData?.result
    },
    currentUser() {
      return this.GetAuthData?.result
    },
  },
  methods: {
    ...mapActions(['logout', 'initAuth', 'GetUnReadedNotifyNum', 'GetUnreadedMessageNum']),

    GoSearch() {
      if (!this.searchText) return
      this.$router.push({ path: '/Search', query: { search: this.searchText } })
    },
    Profile() {
      const id = this.currentUser?._id
      this.$router.push(`/Profile/${id}`)
    },
    async LogUserOut() {
      await this.logout()
      this.$router.push('/Auth')
    },
    GoToNotification() {
      this.$router.push('/Notification')
    },
    GoToChat() {
      this.$router.push('/Chat')
    },

    async loadNavData() {
      const user = this.currentUser
      if (!user) return

      try {
        const notifyList = await this.GetUnReadedNotifyNum(user._id)
        this.notificationNum = notifyList.filter((n) => !n.isReaded).length

        const { total } = await this.GetUnreadedMessageNum(user._id)
        this.unReadedMessages = total
      } catch (error) {
        console.log('Failed to load nav data:', error)
      }
    },
  },
  async mounted() {
    await this.loadNavData()
  },
}
</script>

<style lang="sass">
.nuks
  width: 250px
  text-align: center
  display: inline-block !important

.q-toolbar-title
  display: flex
  align-items: center

.q-btn
  margin-left: 10px
</style>