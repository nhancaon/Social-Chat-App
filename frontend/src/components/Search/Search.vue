<template>
  <q-page class="constrain q-pa-md search-page">
    <div class="row justify-center">
      <div class="col-12 col-sm-10 col-md-8">
        <q-tabs v-model="model" class="text-grey" active-color="primary" indicator-color="primary" align="justify"
          dense narrow-indicator>
          <q-tab name="Posts" label="Posts" />
          <q-tab name="Users" label="Users" />
        </q-tabs>
        <q-separator />

        <div class="search-body">
          <q-inner-loading :showing="isLoading" />

          <template v-if="!isLoading">
            <q-list v-if="model === 'Users' && users.length" bordered separator>
              <q-item v-for="data in users" :key="data._id" clickable v-ripple @click="goUser(data._id)">
                <q-item-section avatar>
                  <q-avatar size="42px">
                    <img v-if="data?.imageUrl" :src="data.imageUrl" />
                    <img v-else src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png" />
                  </q-avatar>
                </q-item-section>
                <q-item-section>
                  <q-item-label lines="1">{{ data?.name }}</q-item-label>
                  <q-item-label caption lines="1">{{ data?.email }}</q-item-label>
                </q-item-section>
              </q-item>
            </q-list>

            <q-list v-else-if="model === 'Posts' && posts.length" bordered separator>
              <q-item v-for="post in posts" :key="post._id" clickable v-ripple @click="goPost(post._id)">
                <q-item-section v-if="post.selectedFile" thumbnail>
                  <img :src="post.selectedFile" class="post-thumb">
                </q-item-section>
                <q-item-section>
                  <q-item-label lines="1">{{ post?.title }}</q-item-label>
                  <q-item-label caption lines="2">{{ post?.message }}</q-item-label>
                </q-item-section>
              </q-item>
            </q-list>

            <div v-else class="text-grey text-center q-pa-xl">
              Không tìm thấy kết quả
            </div>
          </template>
        </div>
      </div>
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
      if (!id) return
      this.$router.push({ path: `/Profile/${id}` })
    },
    goPost(id) {
      if (!id) return
      this.$router.push({ path: `/PostDetail/${id}` })
    }
  },
  mounted() {
    this.getData()
  }
}
</script>

<style scoped>
.search-body {
  position: relative;
  min-height: 120px;
  padding-top: 8px;
}

.post-thumb {
  width: 56px;
  height: 56px;
  object-fit: cover;
  border-radius: 8px;
}

@media (max-width: 599px) {
  .search-page {
    padding: 8px;
  }
}
</style>
