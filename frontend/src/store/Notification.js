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
    async GetUnReadNotifyNum({ commit }, userId) {
      try {
        const { data } = await api.GetNotificationForUser(userId);
        commit('SET_NOTIFICATIONS', data.notifications || []);
        return data.notifications;
      } catch (error) {
        console.error('GetUnReadNotifyNum error:', error);
        throw error;
      }
    },
    async MarkAllNotifyAsReaded({ commit }, userId) {
      try {
        await api.MarkNotificationAsReaded(userId);
        commit('MARK_ALL_READ');
      } catch (error) {
        console.error('MarkAllNotifyAsReaded error:', error);
        throw error;
      }
    }
  }
};

export default Notification;