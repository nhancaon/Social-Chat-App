import * as api from '@/api/index.js'
function initialState() {
  return {
    currentUser: null,
    viewedProfile: null,
    loading: false,
    error: null,
  }
}

const Users = {
  namespaced: true,
  state: initialState(),
  getters: {
    GetUser: (state) => state.currentUser,
    GetViewedProfile: (state) => state.viewedProfile,
  },
  mutations: {
    SET_CURRENT_USER(state, user) {
      state.currentUser = user
    },
    SET_VIEWED_PROFILE(state, user) {
      state.viewedProfile = user
    },
    SET_LOADING(state, value) { state.loading = value },
    SET_ERROR(state, error) { state.error = error },
    RESET_STATE(state) { Object.assign(state, initialState()) },
  },
  actions: {
    async ensureCurrentUserLoaded({ state, commit, rootGetters }) {
      const id = rootGetters['auth/currentUserId']
      if (id && !state.currentUser) {
        try {
          const { data } = await api.fetchUserProfile(id)
          commit('SET_CURRENT_USER', data.user)
        } catch (error) {
          commit('SET_ERROR', error)
        }
      }
    },

    async GetUserFollowersFollowing({ state, commit }) {
      commit('SET_LOADING', true)
      try {
        // luôn dùng currentUser, không bao giờ bị ảnh hưởng bởi việc xem profile khác
        const followers = state.currentUser?.followers || []
        const following = state.currentUser?.following || []
        const uniqueIds = Array.from(new Set([...followers, ...following]))
        if (uniqueIds.length === 0) return []
        const { data } = await api.fetchUsersByIds(uniqueIds)
        return data.users
      } catch (error) {
        commit('SET_ERROR', error)
        throw error
      } finally {
        commit('SET_LOADING', false)
      }
    },

    // dùng khi xem trang Profile của người khác
    // payload có thể là id (string) hoặc { id, page } khi cần phân trang post
    async GetUserByID({ commit }, payload) {
      const { id, page = 1 } = typeof payload === 'string' ? { id: payload } : payload
      try {
        const { data } = await api.fetchUserProfile(id, page)
        commit('SET_VIEWED_PROFILE', data.user)
        return data
      } catch (error) {
        commit('SET_ERROR', error)
        throw error
      }
    },

    async UpdateUserData({ commit }, userData) {
      const { data } = await api.updateUser(userData)
      commit('SET_CURRENT_USER', data.user) // update chính mình sau khi sửa profile
      return data
    },

    async FollowUser(_, profileID) {
      const { data } = await api.following(profileID)
      return data
    },

    async GetTheUserSug(_, id) {
      const { data } = await api.getSuggestedUsers(id)
      return data
    },
  },
}

export default Users