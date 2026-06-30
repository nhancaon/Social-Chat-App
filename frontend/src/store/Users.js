import * as api from '@/api/index.js'

const Users = {
  state: {
    User: null,
    loading: false,
    error: null,
  },
  getters: {
    GetUser: (state) => state.User,
  },
  mutations: {
    SET_USER(state, user) {
      state.User = user
    },
    SET_LOADING(state, value) {
      state.loading = value
    },
    SET_ERROR(state, error) {
      state.error = error
    },
  },
  actions: {
    // lấy danh sách follower + following hợp nhất, kèm thông tin profile
    async GetUserFollowersFollowing({ state, commit }) {
      commit('SET_LOADING', true)
      try {
        const followers = state.User?.followers || []
        const following = state.User?.following || []
        const uniqueIds = Array.from(new Set([...followers, ...following]))

        const userdata = await Promise.all(
          uniqueIds.map(async (uid) => {
            const { data } = await api.fetchUserProfile(uid)
            return {
              _id: data.user._id,
              name: data.user.name,
              imageUrl: data.user.imageUrl,
            }
          })
        )

        return userdata
      } catch (error) {
        commit('SET_ERROR', error)
        throw error
      } finally {
        commit('SET_LOADING', false)
      }
    },

    async GetUserByID({ commit }, id) {
      try {
        const { data } = await api.fetchUserProfile(id)
        commit('SET_USER', data.user)
        return data
      } catch (error) {
        commit('SET_ERROR', error)
        throw error
      }
    },

    async UpdateUserData({ commit }, userData) {
      try {
        const { data } = await api.UpdateUser(userData)
        commit('SET_USER', data.user)
        return data
      } catch (error) {
        commit('SET_ERROR', error)
        throw error
      }
    },

    async FollowUser(_, profileID) {
      const { data } = await api.following(profileID)
      return data
    },

    async GetTheUserSug(_, id) {
      const { data } = await api.getSugUser(id)
      return data
    },
  },
}

export default Users