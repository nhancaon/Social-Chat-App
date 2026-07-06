<template>
  <q-page class="constrain q-pa-md">
    <div class="row q-col-gutter-lg">
      <div class="col-2"></div>
      <div class="col-8 text-center">
        <div class="q-pa-md">
          <q-btn-toggle v-model="model" toggle-color="primary" :options="[
            { label: 'Posts', value: 'Posts' },
            { label: 'Users', value: 'Users' },
          ]" />
        </div>

        <div class="q-pa-md">
          <q-inner-loading :showing="isLoading" />

          <template v-if="!isLoading">
            <q-list v-if="model === 'Users' && users.length" bordered>
              <q-separator />
              <q-item v-for="data in users" :key="data._id" clickable v-ripple @click="goUser(data._id)">
                <q-item-section avatar>
                  <q-avatar>
                    <img v-if="data?.imageUrl" :src="data.imageUrl" />
                    <img v-else src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png" />
                  </q-avatar>
                </q-item-section>
                <q-item-section>{{ data?.name }}</q-item-section>
              </q-item>
            </q-list>

            <q-list v-else-if="model === 'Posts' && posts.length" bordered>
              <q-separator />
              <q-item v-for="post in posts" :key="post._id" clickable v-ripple @click="goPost(post._id)">
                <q-item-section thumbnail>
                  <img :src="post.selectedFile">
                </q-item-section>
                <q-item-section>{{ post?.title }}</q-item-section>
                <q-item-section>{{ post?.message }}</q-item-section>
              </q-item>
            </q-list>

            <div v-else class="text-grey q-pa-md">
              Không tìm thấy kết quả
            </div>
          </template>
        </div>
      </div>
      <div class="col-2"></div>
    </div>
  </q-page>
</template>

<script>
import { mapActions } from 'vuex';

export default {
  name: 'SearchComponent',
  data() {
    return {
      model: 'Posts',
      users: [],
      posts: [],
      isLoading: false
    }
  },
  watch: {
    '$route.query.search': {
      handler() {
        this.getData()
      }
    }
  },
  methods: {
    ...mapActions('posts', ['getPostsUsersBySearch']),
    async getData() {
      const searchTerm = this.$route.query.search

      if (!searchTerm) {
        this.users = []
        this.posts = []
        return
      }

      this.isLoading = true
      try {
        const allData = await this.getPostsUsersBySearch(String(searchTerm))
        // TODO: xác nhận lại field thật từ API - "user" hay "users"
        this.users = allData?.data.users ?? []
        this.posts = allData?.data.posts ?? []
      } catch (error) {
        console.error('getData error:', error)
        this.users = []
        this.posts = []
      } finally {
        this.isLoading = false
      }
    },
    goUser(id) {
      this.$router.push({ path: `/Profile/${id}` })
    },
    goPost(id) {
      this.$router.push({ path: `/PostDetail/${id}` })
    }
  },
  mounted() {
    this.getData()
  }
}
</script>