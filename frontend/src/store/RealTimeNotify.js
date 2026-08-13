function initialState() {
  return {
    socket: null,
    isConnected: false,
    reconnectAttempts: 0,
    reconnectTimer: null,
  };
}

const RealTimeNotify = {
  namespaced: true,
  state: initialState(),
  getters: {
    isConnected: (state) => state.isConnected,
  },
  mutations: {
    SET_SOCKET(state, socket) {
      state.socket = socket;
      state.isConnected = !!socket;
      state.reconnectAttempts = 0;
    },
    CLEAR_SOCKET(state) {
      state.socket = null;
      state.isConnected = false;
    },
    SET_RECONNECT_TIMER(state, timer) {
      state.reconnectTimer = timer;
    },
    CLEAR_RECONNECT_TIMER(state) {
      if (state.reconnectTimer) clearTimeout(state.reconnectTimer);
      state.reconnectTimer = null;
    },
    INCREMENT_RECONNECT_ATTEMPTS(state) {
      state.reconnectAttempts++;
    },
    RESET_STATE(state) {
      if (state.reconnectTimer) clearTimeout(state.reconnectTimer);
      Object.assign(state, initialState());
    }
  },
  actions: {
    async connectToNotifications({ rootGetters, state, commit, dispatch }) {
      const profile = JSON.parse(localStorage.getItem('profile') || 'null');
      const token = profile?.token;
      if (!token) return;

      if (
        state.socket &&
        (state.socket.readyState === WebSocket.OPEN ||
          state.socket.readyState === WebSocket.CONNECTING)
      ) {
        return;
      }
      commit('CLEAR_RECONNECT_TIMER');

      const baseUrl = process.env.VUE_APP_REALTIME_NOTIFICATION_URL;
      const wsUrl = `${baseUrl}${token}`;
      const socket = new WebSocket(wsUrl);

      socket.onopen = () => {
        commit('SET_SOCKET', socket);
      };

      socket.onmessage = (event) => {
        const notification = JSON.parse(event.data);
        commit('notification/ADD_NOTIFICATION', notification, { root: true });
      };

      socket.onclose = () => {
        // Chỉ clear nếu đây đúng là socket đang được lưu trong state
        // (tránh trường hợp socket cũ đóng muộn sau khi đã có socket mới)
        if (state.socket === socket) {
          commit('CLEAR_SOCKET');
          // chỉ tự reconnect nếu người dùng chưa chủ động logout
          if (rootGetters['auth/currentUserId']) {
            dispatch('scheduleReconnect');
          }
        }
      };

      socket.onerror = () => {
        if (state.socket === socket) {
          commit('CLEAR_SOCKET');
        }
      };
    },

    scheduleReconnect({ state, commit, dispatch }) {
      const attempt = state.reconnectAttempts;
      const delay = Math.min(1000 * 2 ** attempt, 30000); // backoff, tối đa 30s
      commit('INCREMENT_RECONNECT_ATTEMPTS');
      const timer = setTimeout(() => {
        dispatch('connectToNotifications');
      }, delay);
      commit('SET_RECONNECT_TIMER', timer);
    },

    disconnectFromNotifications({ state, commit }) {
      if (state.reconnectTimer) clearTimeout(state.reconnectTimer);
      if (state.socket) {
        // gỡ handler trước khi close để tránh onclose cũ trigger reconnect thừa
        state.socket.onclose = null;
        state.socket.onerror = null;
        state.socket.close();
      }
      commit('RESET_STATE');
    }
  }
};

export default RealTimeNotify;