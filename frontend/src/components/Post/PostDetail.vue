<template>
  <q-page class="constrain q-pa-md">
    <div class="row q-col-gutter-lg">
      <div class="col-3"></div>

      <div class="col-6">
        <q-inner-loading :showing="isLoading" />

        <template v-if="!isLoading">
          <div v-if="post">
            <div class="q-pa-md q-gutter-sm" v-if="!editPost">
              <q-btn v-if="isSameUser" color="primary" icon="eva-edit" @click="toggleEdit" label="Edit Post" />
            </div>
            <Post :post="post" :edit-post="editPost" @change-edit="toggleEdit" />
          </div>

          <div v-else class="text-grey text-center q-pa-xl">
            Không tìm thấy bài viết
          </div>
        </template>
      </div>

      <div class="col-3"></div>
    </div>
  </q-page>
</template>

<script>
import Post from './Post.vue';
import { mapActions, mapGetters } from 'vuex';

export default {
  // TODO: xác nhận route name thật trong router config - giữ "PostDetail" nếu router
  // đang dùng đúng path này, hoặc đổi đồng bộ cả router + mọi nơi push route nếu sửa
  name: 'PostDetail',
  data() {
    return {
      editPost: false,
      post: null,
      isSameUser: false,
      isLoading: false
    }
  },
  computed: {
    ...mapGetters('auth', ['GetAuthData'])
  },
  methods: {
    ...mapActions('posts', ['getPost']),
    toggleEdit() {
      this.editPost = !this.editPost
    },
    async fetchPost() {
      this.isLoading = true
      try {
        const { post } = await this.getPost(this.$route.params.id)
        this.post = post ?? null

        const loggedInUserId = this.GetAuthData?.result?._id
        this.isSameUser = !!post && String(post.creator) === String(loggedInUserId)
      } catch (error) {
        console.error('fetchPost error:', error)
        this.post = null
      } finally {
        this.isLoading = false
      }
    }
  },
  mounted() {
    this.fetchPost()
  },
  components: {
    Post
  }
}
</script>