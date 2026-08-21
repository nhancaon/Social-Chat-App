import * as api from '../api/index.js';

const DEFAULT_QUOTA_BYTES = 5 * 1024 * 1024 * 1024 // fallback mirrors backend's DefaultStorageQuotaBytes

function initialState() {
  return {
    isLoading: true,
    files: [],
    storageUsedBytes: 0,
    storageQuotaBytes: DEFAULT_QUOTA_BYTES,
  }
}

const Files = {
  namespaced: true,
  state: initialState(),

  getters: {
    getAllFiles: (state) => state.files,
    isLoading: (state) => state.isLoading,
    storageUsedBytes: (state) => state.storageUsedBytes,
    storageQuotaBytes: (state) => state.storageQuotaBytes,
  },

  mutations: {
    SET_LOADING(state, payload) {
      state.isLoading = payload;
    },
    SET_FILES(state, payload) {
      state.files = payload;
    },
    UPSERT_FILE(state, payload) {
      const index = state.files.findIndex((f) => f._id === payload._id);
      if (index !== -1) {
        state.files.splice(index, 1, { ...state.files[index], ...payload });
      } else {
        state.files.unshift(payload);
      }
    },
    REMOVE_FILE(state, id) {
      state.files = state.files.filter((f) => f._id !== id);
    },
    SET_QUOTA(state, { used, quota }) {
      state.storageUsedBytes = used;
      state.storageQuotaBytes = quota || DEFAULT_QUOTA_BYTES;
    },
    ADD_USED_BYTES(state, bytes) {
      state.storageUsedBytes += bytes;
    },
    RESET_STATE(state) {
      Object.assign(state, initialState())
    },
  },

  actions: {
    async getFiles({ commit }) {
      try {
        commit('SET_LOADING', true);
        const { data } = await api.listFiles();
        commit('SET_FILES', data.data ?? []);
        return data.data;
      } catch (error) {
        console.error('getFiles error:', error);
        throw error;
      } finally {
        commit('SET_LOADING', false);
      }
    },

    // quota isn't returned by /files — it lives on the user document, so this
    // reuses the existing user-profile endpoint instead of a new backend route
    async loadQuota({ commit, rootGetters }) {
      const userId = rootGetters['auth/currentUserId'];
      if (!userId) return;
      try {
        const { data } = await api.fetchUserProfile(userId);
        commit('SET_QUOTA', {
          used: data.user?.storageUsedBytes ?? 0,
          quota: data.user?.storageQuotaBytes,
        });
      } catch (error) {
        console.error('loadQuota error:', error);
      }
    },

    async requestUploadUrl(_, fileMeta) {
      try {
        const { data } = await api.requestUploadUrl(fileMeta);
        return data;
      } catch (error) {
        console.error('requestUploadUrl error:', error);
        throw error;
      }
    },

    async confirmUpload({ commit }, fileId) {
      try {
        const { data } = await api.confirmUpload(fileId);
        commit('UPSERT_FILE', data.data);
        commit('ADD_USED_BYTES', data.data.sizeBytes);
        return data.data;
      } catch (error) {
        console.error('confirmUpload error:', error);
        throw error;
      }
    },

    async getDownloadUrl(_, fileId) {
      try {
        const { data } = await api.getDownloadUrl(fileId);
        return data;
      } catch (error) {
        console.error('getDownloadUrl error:', error);
        throw error;
      }
    },

    async restoreFile({ commit }, fileId) {
      try {
        const { data } = await api.restoreFile(fileId);
        commit('UPSERT_FILE', { _id: fileId, storageClass: data.storageClass });
        return data;
      } catch (error) {
        console.error('restoreFile error:', error);
        throw error;
      }
    },

    async deleteFile({ commit }, fileId) {
      try {
        await api.deleteFile(fileId);
        commit('REMOVE_FILE', fileId);
      } catch (error) {
        console.error('deleteFile error:', error);
        throw error;
      }
    },
  },
};

export default Files;
