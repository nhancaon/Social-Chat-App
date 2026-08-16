<template>
  <div>
    <q-item v-for="user in suggestedUsers" :key="user._id" class="RightBar" :to="`/Profile/${user?._id}`">
      <q-item-section avatar>
        <q-avatar>
          <img v-if="user.imageUrl" :src="user.imageUrl">
          <img v-else :src="defaultAvatar">
        </q-avatar>
      </q-item-section>
      <q-item-section>
        <q-item-label class="text-bold">{{ user?.name }}</q-item-label>
        <q-item-label caption>{{ user?.bio }}</q-item-label>
      </q-item-section>
    </q-item>
  </div>
</template>

<script>
import { mapActions, mapGetters } from 'vuex';

export default {
  name: 'RightBar',
  data() {
    return {
      suggestedUsers: [],
      defaultAvatar: 'https://cdn-icons-png.flaticon.com/512/3237/3237472.png'
    }
  },
  computed: {
    ...mapGetters('auth', ['getAuthData'])
  },
  watch: {
    getAuthData: {
      immediate: true,
      handler(newVal) {
        const loggedInUser = newVal?.result
        if (loggedInUser) {
          this.fetchSuggestedUsers(loggedInUser._id)
        }
      }
    }
  },
  methods: {
    ...mapActions('users', ['getSuggestedUsers']),
    async fetchSuggestedUsers(userId) {
      try {
        const result = await this.getSuggestedUsers(userId)
        this.suggestedUsers = result?.users ?? []
      } catch (error) {
        console.error('fetchSuggestedUsers error:', error)
        this.suggestedUsers = []
      }
    }
  }
}
</script>

<style lang="sass" scoped>
.RightBar
    cursor: pointer
</style>