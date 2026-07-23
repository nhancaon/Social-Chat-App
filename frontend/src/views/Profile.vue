<template>
  <q-page class="constrain q-pa-md">
    <div class="row q-col-gutter-lg constrain">
      <q-inner-loading :showing="isLoading" />

      <template v-if="!isLoading && userData">
        <ShowProfile v-if="!editMode" :userData="userData" :userPosts="userPosts" :isSameUser="isSameUser"
          @edit-profile="toggleEditMode" @update-user="updateUserLocal" />

        <EditProfile v-else :userData="userData" :isSameUser="isSameUser" @edit-profile="toggleEditMode"
          @update-user="updateUserLocal" />

        <div class="col-12">
          <q-separator inset />
        </div>

        <div class="col-4" v-for="post in userPosts" :key="post._id">
          <Post :post="post" />
        </div>
      </template>

      <div v-else-if="!isLoading && !userData" class="col-12 text-center text-grey q-pa-xl">
        Không tìm thấy người dùng
      </div>
    </div>
  </q-page>
</template>

<script>
import { mapGetters, mapActions } from 'vuex';
import Post from '@/components/Post/Post.vue';
import ShowProfile from '@/components/User/ShowProfile.vue'
import EditProfile from '@/components/User/EditProfile.vue';

export default {
  name: 'ProfileView',
  data() {
    return {
      userPosts: [],
      userData: null,
      isSameUser: false,
      editMode: false,
      isLoading: false
    }
  },
  watch: {
    '$route.params.id': {
      handler() {
        this.fetchProfileData()
      }
    }
  },
  computed: {
    ...mapGetters('auth', ['GetAuthData'])
  },
  methods: {
    ...mapActions('users', ['GetUserByID']),
    toggleEditMode() {
      this.editMode = !this.editMode
    },
    updateUserLocal(payload) {
      this.userData = { ...this.userData, ...payload.data }
    },
    async fetchProfileData() {
      this.isLoading = true
      try {
        const loggedInUserId = this.GetAuthData?.result?._id
        const profileId = this.$route.params.id

        const data = await this.GetUserByID(profileId)

        this.userData = data?.user ?? null
        this.userPosts = data?.posts ?? []
        this.isSameUser = String(loggedInUserId) === String(profileId)
      } catch (error) {
        console.error('fetchProfileData error:', error)
        this.userData = null
        this.userPosts = []
      } finally {
        this.isLoading = false
      }
    }
  },
  mounted() {
    this.fetchProfileData()
  },
  components: { ShowProfile, EditProfile, Post }
}
</script>