import * as api from '../api/index.js';

function initialState() {
  return {
    list: [],
    unreadCount: 0,
  };
}

const Notification = {
  namespaced: true,
  state: initialState(),
  getters: {
    getNotifications: (state) => state.list,
    getUnreadNotificationCount: (state) => state.unreadCount,
  },
  mutations: {
    SET_NOTIFICATIONS(state, notifications) {
      state.list = notifications;
      state.unreadCount = notifications.filter(n => !n.isRead).length;
    },
    ADD_NOTIFICATION(state, notification) {
      state.list.unshift(notification);
      if (!notification.isRead) state.unreadCount++;
    },
    MARK_ALL_READ(state) {
      state.list.forEach(n => n.isRead = true);
      state.unreadCount = 0;
    },
    RESET_STATE(state) {
      Object.assign(state, initialState());
    }
  },
  actions: {
    async getUnreadNotifyNum({ commit }, userId) {
      try {
        const { data } = await api.getNotificationForUser(userId);
        commit('SET_NOTIFICATIONS', data.notifications || []);
        return data.notifications;
      } catch (error) {
        console.error('getUnreadNotifyNum error:', error);
        throw error;
      }
    },
    async markAllNotifyAsRead({ commit }, userId) {
      try {
        await api.markNotificationAsRead(userId);
        commit('MARK_ALL_READ');
      } catch (error) {
        console.error('markAllNotifyAsRead error:', error);
        throw error;
      }
    }
  }
};

export default Notification;