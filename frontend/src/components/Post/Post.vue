<template>
  <div>
    <!-- show post -->
    <q-card v-if="!editPost" class="card-post q-mb-md" flat bordered>
      <q-item>
        <q-item-section avatar>
          <q-avatar>
            <img v-if="user?.imageUrl" :src="user.imageUrl" />
            <img v-else src="https://cdn-icons-png.flaticon.com/512/1077/1077063.png" />
          </q-avatar>
        </q-item-section>

        <q-item-section>
          <q-item-label class="text-bold">{{ user?.name }}</q-item-label>
          <q-item-label caption>{{ formattedTime }}</q-item-label>
        </q-item-section>
      </q-item>

      <q-separator />
      <q-img style="cursor: pointer;" @click="goToDetails" :src="localPost.selectedFile" />

      <q-card-section>
        <div class="text-h6">{{ localPost.title }}</div>
        <div class="text-subtitle1">{{ localPost.message }}</div>
        <q-separator />
        <div class="text-subtitle4" v-for="(comment, index) in localPost.comments ?? []" :key="index">
          {{ comment }}
        </div>

        <q-btn @click="like" flat round color="red" :icon="userLike ? 'eva-heart' : 'eva-heart-outline'">
          {{ likesCount }}
        </q-btn>
      </q-card-section>

      <q-input outlined v-model="form.text" label="add comment..">
        <template #append>
          <q-btn v-if="form.text !== ''" @click="addComment" flat round color="primary" icon="eva-plus-square" />
        </template>
      </q-input>
    </q-card>

    <!-- edit post -->
    <div v-else class="q-pa-md items-start q-gutter-md">
      <q-card class="my-card col-12">
        <q-card-section>
          <div class="text-h6">Edit Post</div>
          <q-input dense v-model="localPost.title" autofocus placeholder="Post Title" />
          <div>
            <q-input v-model="localPost.message" placeholder="What's on your mind!" type="textarea" />
          </div>
          <div class="q-pa-md">
            <q-file v-model="file" label="Pick Image" filled />
          </div>

          <div v-if="localPost.selectedFile">
            <q-img :src="localPost.selectedFile" spinner-color="red" style="height: 140px; max-width: 150px;" />
          </div>

          <q-btn flat label="Update" :loading="isUpdating" @click="fireUpdate" />
        </q-card-section>
      </q-card>
    </div>
  </div>
</template>

<script>

import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { mapActions, mapGetters } from 'vuex';
dayjs.extend(relativeTime)

export default {
  name: 'PostComponent',
  props: {
    post: { type: Object, required: true },
    editPost: { type: Boolean, default: false }
  },
  data() {
    return {
      user: {},
      form: { text: '' },
      file: null,
      userLike: false,
      localPost: {},
      isUpdating: false
    }
  },
  watch: {
    file(newVal) {
      if (newVal) this.convertToBase64()
    },
    post: {
      handler(newVal) {
        this.localPost = JSON.parse(JSON.stringify(newVal))
      },
      deep: true
    }
  },
  computed: {
    ...mapGetters('auth', ['GetAuthData']),
    likesCount() {
      return this.localPost.likes?.length ?? 0
    },
    formattedTime() {
      return dayjs(this.localPost?.createdAt).fromNow()
    }
  },
  methods: {
    ...mapActions('users', ['GetUserByID']),
    ...mapActions('posts', ['likePostByUser', 'commentPost', 'updatePost']),

    goToDetails() {
      this.$router.push({ path: `/PostDetail/${this.localPost?._id}` })
    },

    async fireUpdate() {
      this.isUpdating = true
      try {
        const postData = {
          id: this.localPost._id,
          title: this.localPost.title,
          selectedFile: this.localPost.selectedFile,
          message: this.localPost.message,
        }
        const res = await this.updatePost(postData)
        if (res) {
          this.$emit('change-edit')
        }
      } catch (error) {
        console.error('fireUpdate error:', error)
      } finally {
        this.isUpdating = false
      }
    },

    async like() {
      const uid = this.GetAuthData?.result?._id
      if (!uid) return

      const wasLiked = this.userLike
      this.userLike = !wasLiked
      if (wasLiked) {
        this.localPost.likes = (this.localPost.likes ?? []).filter((id) => id !== uid)
      } else {
        this.localPost.likes = [...(this.localPost.likes ?? []), uid]
      }

      try {
        await this.likePostByUser(this.localPost._id)
      } catch (error) {
        console.error('like error:', error)
        // rollback nếu request fail
        this.userLike = wasLiked
        if (wasLiked) {
          this.localPost.likes = [...(this.localPost.likes ?? []), uid]
        } else {
          this.localPost.likes = (this.localPost.likes ?? []).filter((id) => id !== uid)
        }
      }
    },

    async addComment() {
      if (!this.form.text) return
      const commentText = this.form.text
      this.form.text = ''

      try {
        await this.commentPost({ value: commentText, id: this.localPost._id })
        this.localPost.comments = [...(this.localPost.comments ?? []), commentText]
      } catch (error) {
        console.error('addComment error:', error)
        this.form.text = commentText
      }
    },

    convertToBase64() {
      if (!this.file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.localPost.selectedFile = reader.result
      }
      reader.onerror = (error) => console.error('Lỗi đọc file:', error)
      reader.readAsDataURL(this.file)
    }
  },
  async mounted() {
    this.localPost = JSON.parse(JSON.stringify(this.post))

    try {
      const { user } = await this.GetUserByID(this.localPost?.creator)
      this.user = user ?? {}
    } catch (error) {
      console.error('GetUserByID error:', error)
      this.user = {}
    }

    const uid = this.GetAuthData?.result?._id
    const likes = this.localPost.likes ?? []
    this.userLike = !!likes.find((id) => id === uid)
  }
}
</script>