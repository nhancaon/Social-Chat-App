import * as api from '../api/index.js'
import { jwtDecode } from 'jwt-decode'

function initialState() {
  return { authData: null } // { token, result: { _id, name, ... } }
}


const Auth = {
  namespaced: true,
  state: initialState(),
  getters: {
    GetAuthData: (state) => state.authData,
    isAuthenticated: (state) => !!state.authData?.token,
    currentUserId: (state) => state.authData?.result?._id || null,
  },
  mutations: {
    SET_AUTH(state, payload) {
      localStorage.setItem('profile', JSON.stringify({ ...payload }))
      state.authData = payload
    },
    LOGOUT(state) {
      localStorage.removeItem('profile')
      Object.assign(state, initialState())
    },
    RESET_STATE(state) {
      Object.assign(state, initialState())
    },
  },
  actions: {
    async signin({ commit, dispatch }, formData) {
      try {
        const { data } = await api.signIn(formData)
        commit('SET_AUTH', data)
        dispatch('fetchInitialData', data.result._id)
        return data
      } catch (error) {
        console.log(error)
        throw error
      }
    },

    async signup({ commit }, formData) {
      try {
        const { data } = await api.signUp(formData)
        commit('SET_AUTH', data)
        return data
      } catch (error) {
        console.log(error)
        throw error
      }
    },

    // gọi 1 lần lúc app khởi động (vd: App.vue mounted) để khôi phục session từ localStorage
    initAuth({ commit, dispatch }) {
      let user = null
      try {
        user = JSON.parse(localStorage.getItem('profile'))
      } catch (error) {
        console.log('Invalid profile data in localStorage', error)
        commit('LOGOUT')
        return
      }

      const token = user?.token
      if (token) {
        let decoded
        try {
          decoded = jwtDecode(token)
        } catch (error) {
          console.log('Invalid token', error)
          commit('LOGOUT')
          return
        }

        const exp = decoded.exp ?? decoded.expires
        if (exp && exp * 1000 < Date.now()) {
          commit('LOGOUT')
          return
        }
      }

      commit('SET_AUTH', user)
      if (user?.result?._id) {
        dispatch('fetchInitialData', user.result._id)
      }
    },

    async fetchInitialData({ dispatch }, userId) {
      try {
        await Promise.all([
          dispatch('chat/GetUnreadMessageNum', userId, { root: true }),
          dispatch('notification/GetUnReadNotifyNum', userId, { root: true }),
        ])
      } catch (error) {
        // không throw — lỗi fetch unread không nên chặn luồng login/khởi động app
        console.log('fetchInitialData error:', error)
      }
    },

    logout({ commit }) {
      commit('LOGOUT')
      commit('chat/RESET_STATE', null, { root: true })
      commit('notification/RESET_STATE', null, { root: true })
    },
  },
}

export default Auth