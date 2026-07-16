const RealTimeNotify = {
  state: {
    socket: null,
    notificationCount: 0,
    latestNotification: null,
  },
  getters: {
    getNotificationCount: (state) => state.notificationCount,
    getLatestNotification: (state) => state.latestNotification,
  },
  mutations: {
    SET_SOCKET(state, socket) {
      state.socket = socket;
    },
    ADD_NOTIFICATION(state, notification) {
      state.notificationCount += 1;
      state.latestNotification = notification;
    },
    RESET_NOTIFICATION_COUNT(state) {
      state.notificationCount = 0;
    }
  },
  actions: {
    async connectToNotifications(context) {
      const profile = JSON.parse(localStorage.getItem('profile'));

      if (profile && context.state.socket == null) {
        const userId = profile.result._id;
        const baseUrl = process.env.VUE_APP_REALTIME_NOTIFICATION_URL;
        const socket = new WebSocket(`${baseUrl}${userId}`);

        socket.onopen = () => {
          context.commit('SET_SOCKET', socket);
        };

        socket.onmessage = (event) => {
          const notification = JSON.parse(event.data);
          context.commit('ADD_NOTIFICATION', notification);
        };

        socket.onclose = () => {
          context.commit('SET_SOCKET', null);
        };

        socket.onerror = (error) => {
          console.log('WebSocket error', error);
          context.commit('SET_SOCKET', null);
        };
      }
    },

    async disconnectFromNotifications(context) {
      try {
        if (context.state.socket) {
          context.state.socket.close();
        }
        context.commit('SET_SOCKET', null);
      } catch (error) {
        console.log('error', error);
      }
    }
  },
};

export default RealTimeNotify;