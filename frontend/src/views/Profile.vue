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

        <div v-if="isFetchingMore" class="col-12 q-pa-lg text-center">
          <q-spinner-hourglass color="primary" size="3em" />
        </div>

        <div v-if="hasReachedEnd && userPosts.length > 0" class="col-12 q-pa-md text-center text-grey-6">
          <q-icon name="eva-inbox-outline" size="24px" />
          <div class="q-mt-sm">Đã hết bài viết</div>
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
      isLoading: false,
      current: 1,
      max: 0,
      isFetchingMore: false,
      hasReachedEnd: false
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
      this.current = 1
      this.hasReachedEnd = false
      try {
        const loggedInUserId = this.GetAuthData?.result?._id
        const profileId = this.$route.params.id

        const data = await this.GetUserByID({ id: profileId, page: this.current })

        this.userData = data?.user ?? null
        this.userPosts = data?.posts ?? []
        this.max = data?.numberOfPages ?? 0
        this.hasReachedEnd = this.current >= this.max
        this.isSameUser = String(loggedInUserId) === String(profileId)
      } catch (error) {
        console.error('fetchProfileData error:', error)
        this.userData = null
        this.userPosts = []
      } finally {
        this.isLoading = false
      }
    },

    // nối thêm post cũ hơn khi scroll xuống cuối trang
    async fetchMoreUserPosts() {
      if (this.isFetchingMore || this.hasReachedEnd || this.current >= this.max) {
        return
      }

      this.isFetchingMore = true
      this.current++

      try {
        const profileId = this.$route.params.id
        const data = await this.GetUserByID({ id: profileId, page: this.current })
        this.userPosts = [...this.userPosts, ...(data?.posts ?? [])]
        this.hasReachedEnd = this.current >= this.max
      } catch (error) {
        console.error('fetchMoreUserPosts error:', error)
        this.current--
      } finally {
        this.isFetchingMore = false
      }
    },

    handleScroll() {
      const scrollTop = window.pageYOffset || document.documentElement.scrollTop
      const windowHeight = window.innerHeight
      const documentHeight = document.documentElement.scrollHeight

      if (scrollTop + windowHeight >= documentHeight - 200) {
        this.fetchMoreUserPosts()
      }
    }
  },
  mounted() {
    this.fetchProfileData()
    window.addEventListener('scroll', this.handleScroll, { passive: true })
  },
  beforeUnmount() {
    window.removeEventListener('scroll', this.handleScroll)
  },
  components: { ShowProfile, EditProfile, Post }
}
</script>