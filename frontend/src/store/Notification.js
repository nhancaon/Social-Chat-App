import * as api from '../api/index.js'

function initialState() {
  return {
    unreadNotificationCount: 0,
  }
}

const Notification = {
  namespaced: true,
  state: initialState(),
  getters: {
    getUnreadNotificationCount: (state) => state.unreadNotificationCount,
  },
  mutations: {
    SET_UNREAD_NOTIFICATION_COUNT(state, payload) {
      state.unreadNotificationCount = payload
    },
    RESET_STATE(state) {
      Object.assign(state, initialState())
    },
  },
  actions: {
    // id ở đây là userId
    async GetUnReadNotifyNum({ commit }, id) {
      try {
        const { data } = await api.GetNotificationForUser(id)
        const unreadCount = data.notifications.filter(el => !el.isRead).length

        commit('SET_UNREAD_NOTIFICATION_COUNT', unreadCount)
        return data.notifications
      } catch (error) {
        console.error('GetUnReadNotifyNum error:', error)
        throw error
      }
    },

    // id ở đây cũng là userId (API mark TẤT CẢ thông báo của user là đã đọc)
    async MarkAllNotifyAsReaded({ commit }, id) {
      try {
        const { data } = await api.MarkNotificationAsReaded(id)
        commit('SET_UNREAD_NOTIFICATION_COUNT', 0)
        return data
      } catch (error) {
        console.error('MarkAllNotifyAsReaded error:', error)
        throw error
      }
    }
  }
}

export default Notification;