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
      </div>

      <div class="col-3">
        <RightBar />
      </div>
    </div>

    <div class="q-pa-lg flex justify-center fixed-bottom">
      <Add @created="fetchPosts" />
      <q-pagination v-if="isLoaded && max > 0" v-model="current" color="primary" :max="max" :max-pages="5"
        :ellipses="false" :boundary-numbers="false" />
    </div>
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
      isLoaded: false
    }
  },
  watch: {
    current() {
      this.fetchPosts()
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
    async fetchPosts() {
      this.isLoaded = false
      try {
        const data = await this.getPosts(this.current)
        this.posts = data?.data ?? []
        this.max = data?.numberOfPages ?? 0
      } catch (error) {
        console.error('fetchPosts error:', error)
        this.posts = []
      } finally {
        this.isLoaded = true
      }
    }
  },
  mounted() {
    this.fetchPosts()
  },
}
</script>