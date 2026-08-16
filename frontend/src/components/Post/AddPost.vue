<template>
  <q-page-sticky :position="$q.screen.lt.sm ? 'bottom-right' : 'bottom-left'" v-show="getAuthData?.result"
    :offset="$q.screen.lt.sm ? [18, 18] : [18, 60]">
    <div class="q-pa-md q-gutter-sm">
      <q-btn :label="$q.screen.gt.xs ? 'Create Post' : ''" style="cursor: pointer;" icon="eva-plus-circle-outline"
        color="primary" :round="$q.screen.lt.sm" :size="$q.screen.lt.sm ? 'lg' : 'md'" @click="persistent = true" />

      <!-- mobile optimized dialog -->
      <q-dialog v-model="persistent" persistent transition-show="slide-up" transition-hide="slide-down"
        :maximized="$q.screen.lt.sm" :position="$q.screen.lt.sm ? 'bottom' : 'standard'">
        <q-card :style="$q.screen.lt.sm
          ? 'min-height: 80vh; border-radius: 16px 16px 0 0;'
          : 'min-width: 350px;'">
          <q-card-section class="row items-center q-pb-none" v-if="$q.screen.lt.sm">
            <div class="text-h6">Create Post</div>
            <q-space />
            <q-btn flat round dense icon="eva-close-outline" v-close-popup @click="resetForm" />
          </q-card-section>

          <q-card-section v-else>
            <div class="text-h6">Create Post</div>
          </q-card-section>

          <q-card-section class="q-pt-none">
            <q-input dense v-model="post.title" autofocus placeholder="Post Title" class="q-pa-md" />

            <div class="q-pa-md">
              <q-input v-model="post.message" placeholder="What's on your mind?" type="textarea"
                :rows="$q.screen.lt.sm ? 4 : 3" autogrow />
            </div>

            <div class="q-pa-md">
              <q-file v-model="file" label="Pick Image" filled accept="image/*"
                :style="$q.screen.lt.sm ? 'width: 100%;' : 'max-width: 400px;'">
                <template v-slot:prepend>
                  <q-icon name="eva-camera-outline" />
                </template>
              </q-file>
            </div>

            <div class="q-gutter-sm row items-start" v-if="post.selectedFile">
              <q-img :src="post.selectedFile" spinner-color="red" :style="$q.screen.lt.sm
                ? 'height: 200px; max-width: 100%;'
                : 'height: 140px; max-width: 150px;'" class="rounded-borders" />
            </div>
          </q-card-section>

          <q-card-actions v-if="$q.screen.lt.sm" class="q-pa-md">
            <q-btn flat label="Cancel" v-close-popup class="col" size="md" @click="resetForm" />
            <q-btn unelevated label="Create" color="primary" class="col" size="md" :loading="isSubmitting"
              @click="createPostHandler" />
          </q-card-actions>

          <q-card-actions v-else align="right" class="text-primary">
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
  name: 'AddPost',
  data() {
    return {
      persistent: false,
      post: { title: '', message: '', name: '', selectedFile: null },
      file: null,
      isSubmitting: false,
      maxFileSizeMB: 5 // giới hạn 5MB, chỉnh tuỳ nhu cầu / giới hạn backend
    }
  },
  watch: {
    file(newVal) {
      if (newVal && (newVal instanceof File || newVal instanceof Blob)) {
        if (!this.validateFileSize(newVal)) {
          this.file = null
          this.post.selectedFile = null
          return
        }
        this.convertToBase64()
      } else {
        this.post.selectedFile = null
      }
    }
  },
  computed: {
    ...mapGetters('auth', ['getAuthData'])
  },
  methods: {
    ...mapActions('posts', ['createPost']),

    validateFileSize(file) {
      const maxBytes = this.maxFileSizeMB * 1024 * 1024
      if (file.size > maxBytes) {
        this.$q.notify({
          icon: 'eva-alert-circle-outline',
          type: 'negative',
          message: `Image is too large. Max size is ${this.maxFileSizeMB}MB`
        })
        return false
      }
      return true
    },

    convertToBase64() {
      if (!this.file) return

      const reader = new FileReader()
      reader.onload = (e) => {
        const img = new Image()
        img.onload = () => {
          const canvas = document.createElement('canvas')
          const maxWidth = 1080
          const scale = Math.min(1, maxWidth / img.width)

          canvas.width = img.width * scale
          canvas.height = img.height * scale

          const ctx = canvas.getContext('2d')
          ctx.drawImage(img, 0, 0, canvas.width, canvas.height)

          this.post.selectedFile = canvas.toDataURL('image/jpeg', 0.7)
        }
        img.onerror = () => {
          this.$q.notify({
            icon: 'eva-alert-circle-outline',
            type: 'negative',
            message: 'Invalid image file'
          })
          this.post.selectedFile = null
        }
        img.src = e.target.result
      }
      reader.onerror = (error) => {
        console.error('Lỗi đọc file:', error)
        this.$q.notify({
          icon: 'eva-alert-circle-outline',
          type: 'negative',
          message: 'Error reading file'
        })
        this.post.selectedFile = null
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
      this.post.name = this.getAuthData?.result?.name

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
  }
}
</script>