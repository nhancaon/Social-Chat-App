<template>
  <div class="row col-12 constrain">
    <div class="col-4 text-center">
      <q-avatar size="150px">
        <img v-if="userData?.imageUrl" :src="userData.imageUrl">
        <img v-else src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png">
      </q-avatar>
    </div>
    <div class="col-8 text-left">
      <div class="text-h6 q-pa-lg" style="margin: auto;">
        {{ userData?.name }}
        <q-btn v-if="isSameUser" @click="edit" flat label="Edit" />

        <!-- follow / unfollow -->
        <q-btn v-if="!isSameUser && !isUserFollowing" @click="followOrUnfollow" :loading="isProcessingFollow"
          :disable="isProcessingFollow" flat style="color: #FF0080" label="Follow" />

        <q-btn v-if="!isSameUser && isUserFollowing" @click="followOrUnfollow" :loading="isProcessingFollow"
          :disable="isProcessingFollow" flat class="primary" label="Unfollow" />
      </div>
      <q-separator inset />
      <div class="text-subtitle1 q-pa-lg" style="margin: auto;">
        {{ userData?.bio }}
        <div>
          <i>{{ userPosts.length }} Posts</i>
          <i>{{ userData?.followers?.length || 0 }} followers</i>
          <i>{{ userData?.following?.length || 0 }} following</i>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapActions } from 'vuex';

export default {
  props: {
    userData: { type: Object, required: true },
    userPosts: { type: Array, default: () => [] },
    isSameUser: { type: Boolean, default: false }
  },
  data() {
    return {
      isUserFollowing: false,
      isProcessingFollow: false
    }
  },
  watch: {
    userData() {
      this.checkUserFollowing()
    }
  },
  methods: {
    ...mapActions('users', ['FollowUser', 'GetUserByID']),

    checkUserFollowing() {
      const loggedUserId = JSON.parse(localStorage.getItem('profile'))?.result?._id
      this.isUserFollowing = !!this.userData?.followers?.find((id) => id === loggedUserId)
    },

    async followOrUnfollow() {
      if (this.isProcessingFollow) return
      this.isProcessingFollow = true
      try {
        const data = await this.FollowUser(this.userData._id)
        if (data?.FirstUser) {
          this.$emit('update-user', { data: data.FirstUser })
          this.checkUserFollowing()
        }
      } catch (error) {
        console.error('followOrUnfollow error:', error)
        // TODO: hiển thị thông báo lỗi cho người dùng
      } finally {
        this.isProcessingFollow = false
      }
    },

    edit() {
      this.$emit('edit-profile')
    },
  },
  mounted() {
    this.checkUserFollowing()
  }
}
</script>