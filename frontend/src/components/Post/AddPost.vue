<template>
  <q-page-sticky position="bottom-left" v-show="GetAuthData?.result">
    <div class="q-pa-md q-gutter-sm">
      <q-btn label="Create Post" style="cursor: pointer;" icon="eva-plus-circle-outline" color="primary"
        @click="persistent = true" />

      <q-dialog v-model="persistent" persistent transition-show="scale" transition-hide="scale">
        <q-card style="min-width: 350px;">
          <q-card-section>
            <div class="text-h6">Create Post</div>
          </q-card-section>

          <q-card-section class="q-pt-none">
            <q-input dense v-model="post.title" autofocus placeholder="Post Title" />
            <div class="q-pa-md" style="max-width: 300px;">
              <q-input v-model="post.message" placeholder="What's on your mind?" type="textarea" />
            </div>
            <div class="q-pa-md">
              <q-file v-model="file" label="Pick Image" filled style="max-width: 400px;" />
            </div>

            <div v-if="post.selectedFile" class="q-gutter-sm row items-start">
              <q-img :src="post.selectedFile" spinner-color="red" style="height: 140px; max-width: 150px;" />
            </div>
          </q-card-section>

          <q-card-actions align="right" class="text-primary">
            <q-btn flat label="Cancel" v-close-popup @click="resetForm" />
            <q-btn flat label="Create" :loading="isSubmitting" @click="createPostHandler" />
          </q-card-actions>
        </q-card>
      </q-dialog>
    </div>
  </q-page-sticky>
</template>

<script>
import { mapActions, mapGetters } from 'vuex'

export default {
  name: 'AddComponent',
  data() {
    return {
      persistent: false,
      post: { title: '', message: '', name: '', selectedFile: null },
      file: null,
      isSubmitting: false
    }
  },
  watch: {
    file(newVal) {
      if (newVal) this.convertToBase64()
    }
  },
  computed: {
    ...mapGetters('auth', ['GetAuthData'])
  },
  methods: {
    ...mapActions('posts', ['createPost']),
    convertToBase64() {
      if (!this.file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.post.selectedFile = reader.result
      }
      reader.onerror = (error) => {
        console.error('Lỗi đọc file:', error)
      }
      reader.readAsDataURL(this.file)
    },
    validatePost() {
      const errors = []
      if (!this.post.title) errors.push('Title is required')
      if (!this.post.message) errors.push('Message is required')
      if (!this.post.name) errors.push('User name is missing — please log in again')

      errors.forEach((msg) => {
        this.$q.notify({
          icon: 'eva-alert-circle-outline',
          type: 'negative',
          message: msg
        })
      })

      return errors.length === 0
    },
    resetForm() {
      this.post = { title: '', message: '', name: '', selectedFile: null }
      this.file = null
    },
    async createPostHandler() {
      this.post.name = this.GetAuthData?.result?.name

      if (!this.validatePost()) return

      this.isSubmitting = true
      try {
        const data = await this.createPost(this.post)
        if (data) {
          this.$emit('created')
          this.$q.notify({
            icon: 'eva-alert-circle-outline',
            type: 'positive',
            message: 'Post Created Successfully'
          })
          this.persistent = false
          this.resetForm()
        }
      } catch (error) {
        console.error('createPostHandler error:', error)
        this.$q.notify({
          icon: 'eva-alert-circle-outline',
          type: 'negative',
          message: 'Failed to create post, please try again'
        })
      } finally {
        this.isSubmitting = false
      }
    }
  },
}
</script>