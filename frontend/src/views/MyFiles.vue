<template>
  <q-page class="constrain q-pa-md">
    <div class="row items-center q-mb-lg q-gutter-sm">
      <div class="col-grow">
        <div class="text-h5 text-weight-bold">My Files</div>
        <div class="text-caption text-grey-7">Upload, archive, and restore your files</div>
      </div>
      <q-btn unelevated rounded color="primary" icon="eva-cloud-upload-outline" label="Upload file"
        @click="openUploadDialog" />
    </div>

    <q-card flat bordered class="quota-card q-pa-md q-mb-lg">
      <div class="row items-center q-gutter-md">
        <q-icon name="eva-pie-chart-2-outline" color="primary" size="26px" />
        <div class="col">
          <div class="row items-center q-mb-xs">
            <div class="text-body2 text-weight-medium">
              {{ formatBytes(storageUsedBytes) }} <span class="text-grey-6">of</span> {{ formatBytes(storageQuotaBytes) }} used
            </div>
            <q-space />
            <q-btn flat dense round icon="eva-refresh-outline" color="grey-7" :loading="isLoading" @click="refreshAll" />
          </div>
          <q-linear-progress :value="quotaRatio" :color="quotaRatio > 0.9 ? 'negative' : 'primary'" size="8px" rounded />
        </div>
      </div>
    </q-card>

    <q-card v-if="!isLoading && files.length > 0" flat bordered class="files-card">
      <q-item v-for="(file, index) in files" :key="file._id" class="file-row"
        :class="{ 'file-row--alt': index % 2 === 1 }">
        <q-item-section avatar>
          <div class="file-icon-wrap" :class="`bg-${tierMeta(file.storageClass).color}-1`">
            <q-icon :name="tierMeta(file.storageClass).icon" :color="tierMeta(file.storageClass).color" size="22px" />
          </div>
        </q-item-section>

        <q-item-section>
          <q-item-label class="text-weight-medium ellipsis">{{ file.filename }}</q-item-label>
          <q-item-label caption>
            {{ formatBytes(file.sizeBytes) }} · uploaded {{ formatDate(file.uploadedAt) }}
          </q-item-label>
        </q-item-section>

        <q-item-section side>
          <q-chip dense outline rounded :color="tierMeta(file.storageClass).color" size="sm">
            {{ tierMeta(file.storageClass).label }}
          </q-chip>
        </q-item-section>

        <q-item-section side>
          <div class="row q-gutter-xs">
            <q-btn v-if="canDownload(file)" flat dense round icon="eva-download-outline" color="primary"
              :loading="downloadingId === file._id" @click="handleDownload(file)">
              <q-tooltip>Download</q-tooltip>
            </q-btn>
            <q-btn v-else-if="file.storageClass === 'GLACIER'" flat dense round icon="eva-flip-2-outline"
              color="blue-grey" :loading="restoringId === file._id" @click="handleRestore(file)">
              <q-tooltip>Restore from cold storage</q-tooltip>
            </q-btn>
            <q-btn flat dense round icon="eva-trash-2-outline" color="grey-7" @click="confirmDeleteFile(file)">
              <q-tooltip>Move to trash</q-tooltip>
            </q-btn>
          </div>
        </q-item-section>
      </q-item>
    </q-card>

    <div v-else-if="!isLoading" class="empty-state text-center q-pa-xl">
      <q-icon name="eva-folder-outline" color="grey-4" size="64px" />
      <div class="text-subtitle1 text-grey-7 q-mt-md">No files yet</div>
      <div class="text-caption text-grey-6 q-mb-md">Upload your first file to get started</div>
      <q-btn unelevated rounded color="primary" icon="eva-cloud-upload-outline" label="Upload file"
        @click="openUploadDialog" />
    </div>

    <div v-else class="q-pa-xl text-center">
      <q-spinner-hourglass color="primary" size="3em" />
    </div>

    <!-- upload dialog -->
    <q-dialog v-model="uploadDialog" persistent>
      <q-card class="dialog-card" style="min-width: 380px;">
        <q-card-section class="row items-center q-gutter-sm">
          <q-icon name="eva-cloud-upload-outline" color="primary" size="24px" />
          <div class="text-h6">Upload file</div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <q-file v-model="pickedFile" label="Choose file" filled :disable="isUploading">
            <template v-slot:prepend>
              <q-icon name="eva-attach-outline" />
            </template>
          </q-file>

          <div v-if="isUploading" class="q-mt-md">
            <q-linear-progress :value="uploadProgress" color="primary" size="8px" rounded />
            <div class="text-caption text-grey q-mt-xs">{{ Math.round(uploadProgress * 100) }}%</div>
          </div>
        </q-card-section>

        <q-card-actions align="right" class="q-pa-md">
          <q-btn flat rounded label="Cancel" v-close-popup :disable="isUploading" @click="resetUpload" />
          <q-btn unelevated rounded color="primary" label="Upload" :loading="isUploading" :disable="!pickedFile"
            @click="handleUpload" />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <!-- delete confirm dialog -->
    <q-dialog v-model="deleteDialog">
      <q-card class="dialog-card" style="min-width: 340px;">
        <q-card-section class="row items-center q-gutter-sm">
          <q-icon name="eva-trash-2-outline" color="negative" size="24px" />
          <div class="text-h6">Move to trash?</div>
        </q-card-section>
        <q-card-section class="q-pt-none text-grey-8">
          "{{ fileToDelete?.filename }}" will stay in trash for 30 days before being permanently deleted.
        </q-card-section>
        <q-card-actions align="right" class="q-pa-md">
          <q-btn flat rounded label="Cancel" v-close-popup />
          <q-btn unelevated rounded color="negative" label="Move to trash" @click="handleDeleteConfirmed" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<script>
import axios from 'axios'
import { mapGetters, mapActions } from 'vuex'

// tiers mirror backend/api/models/file_model.go's StorageClass constants
// colors use Quasar's shaded palette names (not primary/secondary) so the
// dynamic `bg-${color}-1` soft-tint class on the file icon resolves correctly
const TIER_META = {
  STANDARD: { label: 'Ready', color: 'blue', icon: 'eva-file-outline' },
  GLACIER: { label: 'Cold storage', color: 'blue-grey', icon: 'eva-cube-outline' },
  RESTORING: { label: 'Restoring...', color: 'orange', icon: 'eva-loader-outline' },
  RESTORED: { label: 'Restored', color: 'teal', icon: 'eva-checkmark-circle-outline' },
}

export default {
  name: 'MyFiles',
  data() {
    return {
      uploadDialog: false,
      pickedFile: null,
      isUploading: false,
      uploadProgress: 0,
      deleteDialog: false,
      fileToDelete: null,
      downloadingId: null,
      restoringId: null,
    }
  },
  computed: {
    ...mapGetters('files', ['getAllFiles', 'isLoading', 'storageUsedBytes', 'storageQuotaBytes']),
    files() {
      return this.getAllFiles
    },
    quotaRatio() {
      if (!this.storageQuotaBytes) return 0
      return Math.min(1, this.storageUsedBytes / this.storageQuotaBytes)
    },
  },
  methods: {
    ...mapActions('files', [
      'getFiles', 'loadQuota', 'requestUploadUrl', 'confirmUpload',
      'getDownloadUrl', 'restoreFile', 'deleteFile',
    ]),

    async refreshAll() {
      await Promise.all([this.getFiles(), this.loadQuota()])
    },

    tierMeta(storageClass) {
      return TIER_META[storageClass] ?? TIER_META.STANDARD
    },

    canDownload(file) {
      return file.storageClass === 'STANDARD' || file.storageClass === 'RESTORED'
    },

    formatBytes(bytes) {
      if (!bytes) return '0 B'
      const units = ['B', 'KB', 'MB', 'GB', 'TB']
      let i = 0
      let val = bytes
      while (val >= 1024 && i < units.length - 1) {
        val /= 1024
        i++
      }
      return `${val.toFixed(val < 10 && i > 0 ? 1 : 0)} ${units[i]}`
    },

    formatDate(dateStr) {
      if (!dateStr) return ''
      return new Date(dateStr).toLocaleDateString('en-US')
    },

    openUploadDialog() {
      this.resetUpload()
      this.uploadDialog = true
    },

    resetUpload() {
      this.pickedFile = null
      this.uploadProgress = 0
    },

    async handleUpload() {
      if (!this.pickedFile) return
      this.isUploading = true
      this.uploadProgress = 0
      try {
        const meta = {
          filename: this.pickedFile.name,
          sizeBytes: this.pickedFile.size,
          contentType: this.pickedFile.type || 'application/octet-stream',
        }
        const { data: fileRecord, uploadUrl } = await this.requestUploadUrl(meta)

        // plain axios, NOT the app's API instance — this PUT goes straight to S3,
        // a different origin that must not see our JWT or the backend baseURL
        await axios.put(uploadUrl, this.pickedFile, {
          headers: { 'Content-Type': meta.contentType },
          onUploadProgress: (evt) => {
            this.uploadProgress = evt.total ? evt.loaded / evt.total : 0
          },
        })

        await this.confirmUpload(fileRecord._id)

        this.$q.notify({ icon: 'eva-alert-circle-outline', type: 'positive', message: 'File uploaded successfully' })
        this.uploadDialog = false
        this.resetUpload()
      } catch (error) {
        console.error('handleUpload error:', error)
        this.$q.notify({ icon: 'eva-alert-circle-outline', type: 'negative', message: 'Upload failed, please try again' })
      } finally {
        this.isUploading = false
      }
    },

    async handleDownload(file) {
      this.downloadingId = file._id
      try {
        const { downloadUrl } = await this.getDownloadUrl(file._id)
        window.open(downloadUrl, '_blank')
      } catch (error) {
        const status = error?.response?.status
        const message = status === 409
          ? 'File is in cold storage — click "Restore" before downloading'
          : 'Failed to download file, please try again'
        this.$q.notify({ icon: 'eva-alert-circle-outline', type: 'negative', message })
      } finally {
        this.downloadingId = null
      }
    },

    async handleRestore(file) {
      this.restoringId = file._id
      try {
        const data = await this.restoreFile(file._id)
        this.$q.notify({
          icon: 'eva-alert-circle-outline',
          type: 'info',
          message: data.message || 'Restore requested — you\'ll be notified when the file is ready',
        })
      } catch (error) {
        console.error('handleRestore error:', error)
        this.$q.notify({ icon: 'eva-alert-circle-outline', type: 'negative', message: 'Restore request failed' })
      } finally {
        this.restoringId = null
      }
    },

    confirmDeleteFile(file) {
      this.fileToDelete = file
      this.deleteDialog = true
    },

    async handleDeleteConfirmed() {
      if (!this.fileToDelete) return
      try {
        await this.deleteFile(this.fileToDelete._id)
        this.$q.notify({ icon: 'eva-alert-circle-outline', type: 'positive', message: 'Moved to trash' })
      } catch (error) {
        console.error('handleDeleteConfirmed error:', error)
        this.$q.notify({ icon: 'eva-alert-circle-outline', type: 'negative', message: 'Failed to delete, please try again' })
      } finally {
        this.deleteDialog = false
        this.fileToDelete = null
      }
    },
  },
  async mounted() {
    await this.refreshAll()
  },
}
</script>

<style scoped>
.quota-card,
.files-card,
.dialog-card {
  border-radius: 12px;
}

.file-row {
  padding-top: 10px;
  padding-bottom: 10px;
}

.file-row--alt {
  background-color: rgba(0, 0, 0, 0.02);
}

.file-icon-wrap {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-state {
  border: 1px dashed rgba(0, 0, 0, 0.12);
  border-radius: 12px;
}
</style>
