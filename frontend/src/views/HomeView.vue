<template>
  <q-page class="constrain q-pa-md">
    <div class="row q-col-gutter-lg">
      <div class="col-3">
        <SideBar />
      </div>

      <div v-if="!isLoaded" class="col-6 q-mx-auto">
        <div class="q-pa-md">
          <q-card>
            <q-item>
              <q-item-section avatar>
                <q-skeleton type="QAvatar" />
              </q-item-section>

              <q-item-section>
                <q-item-label>
                  <q-skeleton type="text" />
                </q-item-label>
                <q-item-label caption>
                  <q-skeleton type="text" />
                </q-item-label>
              </q-item-section>
            </q-item>

            <q-skeleton height="200px" square />
            <q-card-actions class="q-gutter-md">
              <q-skeleton type="QBtn" />
              <q-skeleton type="QBtn" />
            </q-card-actions>
          </q-card>
        </div>
      </div>

      <div v-else class="col-6 q-mx-auto">
        <div v-if="posts.length === 0" class="text-grey text-center q-pa-xl">
          Chưa có bài viết nào
        </div>

        <Post v-for="post in posts" :key="post._id" :post="post" />

        <div v-if="isFetchingMore" class="q-pa-lg text-center">
          <q-spinner-hourglass color="primary" size="3em" />
          <div class="q-mt-md text-grey-7">Đang tải thêm bài viết...</div>
        </div>

        <div v-if="hasReachedEnd && posts.length > 0" class="q-pa-md text-center text-grey-6">
          <q-icon name="eva-inbox-outline" size="24px" />
          <div class="q-mt-sm">Đã hết bài viết</div>
        </div>

        <div class="bottom-spacer"></div>
      </div>

      <div class="col-3">
        <RightBar />
      </div>
    </div>

    <Add @created="onPostCreated" />
  </q-page>
</template>

<script>
import Add from '@/components/Post/AddPost.vue'
import Post from '@/components/Post/Post.vue'
import SideBar from '@/components/SideBar/SideBar.vue';
import RightBar from '@/components/RightBar/RightBar.vue';
import { mapActions } from 'vuex';

export default {
  name: 'HomeView',
  data() {
    return {
      current: 1,
      max: 0,
      posts: [],
      isLoaded: false,
      isFetchingMore: false,
      hasReachedEnd: false
    }
  },
  components: {
    Add,
    Post,
    SideBar,
    RightBar,
  },
  methods: {
    ...mapActions('posts', ['getPosts']),

    // Load trang đầu tiên hoặc reset danh sách
    async fetchPosts() {
      this.isLoaded = false
      try {
        const data = await this.getPosts(this.current)
        this.posts = data?.data ?? []
        this.max = data?.numberOfPages ?? 0
        this.hasReachedEnd = this.current >= this.max
      } catch (error) {
        console.error('fetchPosts error:', error)
        this.posts = []
        this.hasReachedEnd = false
      } finally {
        this.isLoaded = true
      }
    },

    // Nối thêm bài viết khi scroll xuống cuối
    async fetchMorePosts() {
      if (this.isFetchingMore || this.hasReachedEnd || this.current >= this.max) {
        return
      }

      this.isFetchingMore = true
      this.current++

      try {
        const data = await this.getPosts(this.current)
        this.posts = [...this.posts, ...(data?.data ?? [])]
        this.hasReachedEnd = this.current >= this.max
      } catch (error) {
        console.error('fetchMorePosts error:', error)
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
        this.fetchMorePosts()
      }
    },

    async onPostCreated() {
      this.current = 1
      this.hasReachedEnd = false
      await this.fetchPosts()
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  },
  async mounted() {
    await this.fetchPosts()
    window.addEventListener('scroll', this.handleScroll, { passive: true })
  },
  beforeUnmount() {
    window.removeEventListener('scroll', this.handleScroll)
  }
}
</script>

<style scoped>
.q-page {
  scroll-behavior: smooth;
  position: relative;
  min-height: 100vh;
}
</style>