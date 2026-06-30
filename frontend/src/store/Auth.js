import * as api from '../api/index.js'
import { jwtDecode } from 'jwt-decode'

const Auth = {
  state: {
    authData: null,
  },
  getters: {
    GetAuthData: (state) => state.authData,
  },
  mutations: {
    SET_AUTH(state, payload) {
      localStorage.setItem('profile', JSON.stringify({ ...payload }))
      state.authData = payload
    },
    SET_AUTH_FROM_STORAGE(state, user) {
      state.authData = user
    },
    LOGOUT(state) {
      localStorage.removeItem('profile')
      state.authData = null
    },
  },
  actions: {
    async signin({ commit }, formData) {
      try {
        const { data } = await api.signIn(formData)
        commit('SET_AUTH', data)
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
    initAuth({ commit }) {
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

      commit('SET_AUTH_FROM_STORAGE', user)
    },

    logout({ commit }) {
      commit('LOGOUT')
    },
  },
}

export default Auth