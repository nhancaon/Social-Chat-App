<template>
  <div class="row col-12 constrain">
    <div class="col-4 text-center">
      <q-avatar size="150px">
        <img :src="localUserData?.imageUrl">
      </q-avatar>
    </div>
    <div class="col-8 text-left">
      <i>Edit Profile</i>
      <div class="text-h6 q-pa-lg" style="margin: auto;">
        <q-btn v-if="isSameUser" @click="save" :loading="isSaving" :disable="isSaving" flat label="Save" />
      </div>

      <q-input dense v-model="localUserData.name" autofocus placeholder="User Data Title" />
      <div>
        <q-input v-model="localUserData.bio" placeholder="What's On Your Mind? Your Bio" type="textarea" />
      </div>
      <div class="q-pa-md">
        <q-file v-model="file" label="Pick Image" filled />
      </div>
    </div>
  </div>
</template>

<script>
import { mapActions } from 'vuex';

export default {
  props: {
    userData: {
      type: Object,
      required: true
    },
    isSameUser: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      file: null,
      localUserData: { ...this.userData },
      isSaving: false
    }
  },
  watch: {
    file(newVal) {
      if (newVal) this.convertToBase64()
    },
    userData: {
      handler(newVal) {
        this.localUserData = { ...newVal }
      },
      deep: true
    }
  },
  methods: {
    ...mapActions('users', ['updateUserData']),
    convertToBase64() {
      if (!this.file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.localUserData.imageUrl = reader.result
      }
      reader.onerror = (error) => {
        console.error('Lỗi đọc file:', error)
      }
      reader.readAsDataURL(this.file)
    },

    async save() {
      if (this.isSaving) return
      this.isSaving = true
      try {
        const userdata = {
          _id: this.localUserData._id,
          name: this.localUserData.name,
          bio: this.localUserData.bio,
          imageUrl: this.localUserData.imageUrl
        }

        const update = await this.updateUserData(userdata)

        if (update?.data) {
          this.$emit('update-user', { data: update.data })
          this.$emit('edit-profile')
        }
      } catch (error) {
        console.error('Save profile error:', error)
      } finally {
        this.isSaving = false
      }
    }
  },
}
</script>