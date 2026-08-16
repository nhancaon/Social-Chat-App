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
      <q-img v-if="localPost?.selectedFile" style="cursor: pointer;" @click="goToDetails" :src="localPost.selectedFile" />

      <q-card-section>
        <div class="text-h6">{{ localPost.title }}</div>
        <div class="text-subtitle1">{{ localPost.message }}</div>

        <div class="row items-center q-mt-sm">
          <q-btn @click="like" flat round color="red" :icon="userLike ? 'eva-heart' : 'eva-heart-outline'">
            {{ likesCount }}
          </q-btn>
        </div>

        <q-separator class="q-my-md" />

        <div class="comments-section">
          <div class="text-subtitle2 text-grey-8 q-mb-sm">Comments</div>

          <div
            v-for="comment in displayedComments"
            :key="comment._id"
            class="comment-item q-mb-sm"
          >
            <q-item dense>
              <q-item-section avatar>
                <q-avatar size="32px">
                  <img v-if="comment.user?.imageUrl" :src="comment.user.imageUrl" />
                  <img v-else src="https://cdn-icons-png.flaticon.com/512/1077/1077063.png" />
                </q-avatar>
              </q-item-section>

              <q-item-section>
                <q-item-label class="text-bold text-caption">{{ comment.user?.name ?? 'Unknown user' }}</q-item-label>
                <q-item-label class="text-body2">{{ comment.value }}</q-item-label>
                <q-item-label caption>{{ getCommentTime(comment.createdAt) }}</q-item-label>
              </q-item-section>

              <q-item-section side v-if="canDeleteComment(comment)">
                <q-btn
                  flat round dense size="sm" color="grey"
                  icon="eva-close-outline"
                  :loading="deletingComments[comment._id]"
                  @click="deleteCommentConfirm(comment._id)"
                />
              </q-item-section>
            </q-item>
          </div>

          <div v-if="hasMoreComments" class="text-center q-mb-sm">
            <q-btn
              v-if="!showAllComments"
              flat dense color="primary" class="text-caption"
              @click="showAllComments = true"
            >
              Show {{ remainingCommentsCount }} more comments
            </q-btn>
            <q-btn
              v-else
              flat dense color="primary" class="text-caption"
              @click="showAllComments = false"
            >
              Show less
            </q-btn>
          </div>

          <div v-if="!localPost.comments || localPost.comments.length === 0" class="text-grey-6 text-center q-py-md">
            No comments yet. Be the first to comment!
          </div>
        </div>
      </q-card-section>

      <q-separator />

      <q-card-section>
        <q-input outlined dense v-model="form.text" label="Add a comment..." :loading="isCommenting" @keyup.enter="addComment">
          <template #append>
            <q-btn v-if="form.text !== ''" @click="addComment" flat round color="primary" icon="eva-plus-square" :loading="isCommenting" />
          </template>
        </q-input>
      </q-card-section>
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
      isUpdating: false,
      isCommenting: false,
      showAllComments: false,
      deletingComments: {}
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
    ...mapGetters('auth', ['getAuthData']),
    likesCount() {
      return this.localPost.likes?.length ?? 0
    },
    formattedTime() {
      return dayjs(this.localPost?.createdAt).fromNow()
    },
    displayedComments() {
      const comments = this.localPost.comments ?? []
      if (this.showAllComments || comments.length <= 2) return comments
      return comments.slice(0, 2)
    },
    hasMoreComments() {
      return (this.localPost.comments?.length ?? 0) > 2
    },
    remainingCommentsCount() {
      return Math.max((this.localPost.comments?.length ?? 0) - 2, 0)
    }
  },
  methods: {
    ...mapActions('users', ['getUserById']),
    ...mapActions('posts', ['likePost', 'commentPost', 'deleteComment', 'updatePost']),

    canDeleteComment(comment) {
      const uid = this.getAuthData?.result?._id
      if (!uid) return false
      return comment.userId === uid || this.localPost.creator === uid
    },

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
      const uid = this.getAuthData?.result?._id
      if (!uid) return

      const wasLiked = this.userLike
      this.userLike = !wasLiked
      if (wasLiked) {
        this.localPost.likes = (this.localPost.likes ?? []).filter((id) => id !== uid)
      } else {
        this.localPost.likes = [...(this.localPost.likes ?? []), uid]
      }

      try {
        await this.likePost(this.localPost._id)
      } catch (error) {
        console.error('like error:', error)
        // rollback if the request fails
        this.userLike = wasLiked
        if (wasLiked) {
          this.localPost.likes = [...(this.localPost.likes ?? []), uid]
        } else {
          this.localPost.likes = (this.localPost.likes ?? []).filter((id) => id !== uid)
        }
      }
    },

    getCommentTime(createdAt) {
      return createdAt ? dayjs(createdAt).fromNow() : 'Just now'
    },

    async addComment() {
      if (!this.form.text) return
      const commentText = this.form.text
      this.form.text = ''

      this.isCommenting = true
      try {
        const updatedPost = await this.commentPost({ value: commentText, id: this.localPost._id })
        this.localPost.comments = updatedPost?.comments ?? this.localPost.comments
      } catch (error) {
        console.error('addComment error:', error)
        this.form.text = commentText
        this.$q.notify({ color: 'negative', message: 'Failed to add comment', icon: 'error' })
      } finally {
        this.isCommenting = false
      }
    },

    // just asks for confirmation here; the actual delete happens in deleteCommentNow
    deleteCommentConfirm(commentId) {
      this.$q.dialog({
        title: 'Delete comment',
        message: 'Are you sure you want to delete this comment?',
        persistent: true,
        ok: { label: 'Delete', color: 'negative', flat: true },
        cancel: { label: 'Cancel', flat: true }
      }).onOk(() => {
        this.deleteCommentNow(commentId)
      })
    },

    async deleteCommentNow(commentId) {
      this.deletingComments = { ...this.deletingComments, [commentId]: true }
      try {
        const updatedPost = await this.deleteComment({ postId: this.localPost._id, commentId })
        this.localPost.comments = updatedPost?.comments ?? this.localPost.comments
        this.$q.notify({ color: 'positive', message: 'Comment deleted' })
      } catch (error) {
        console.error('deleteComment error:', error)
        this.$q.notify({ color: 'negative', message: 'Failed to delete comment', icon: 'error' })
      } finally {
        const rest = { ...this.deletingComments }
        delete rest[commentId]
        this.deletingComments = rest
      }
    },

    convertToBase64() {
      if (!this.file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.localPost.selectedFile = reader.result
      }
      reader.onerror = (error) => console.error('File read error:', error)
      reader.readAsDataURL(this.file)
    }
  },
  async mounted() {
    this.localPost = JSON.parse(JSON.stringify(this.post))

    try {
      const { user } = await this.getUserById(this.localPost?.creator)
      this.user = user ?? {}
    } catch (error) {
      console.error('getUserById error:', error)
      this.user = {}
    }

    const uid = this.getAuthData?.result?._id
    const likes = this.localPost.likes ?? []
    this.userLike = !!likes.find((id) => id === uid)
  }
}
</script>

<style scoped>
.card-post {
  border-radius: 12px;
}
.comments-section {
  max-height: 320px;
  overflow-y: auto;
}
.comment-item {
  background-color: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
}
</style>
