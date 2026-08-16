function initialState() {
  return {
    ws: null,
    privateMessages: [],
    onlineFriends: [],
    userId: '',
    // Increments on every incoming message. Watching this primitive is
    // 100% reliable in Vue 3, unlike deep-watching a nested reactive array
    // through a getter, which can silently miss updates in some cases.
    messageReceivedTick: 0,
  };
}

const RealTimeChat = {
  namespaced: true,
  state: initialState(),
  getters: {
    getUserId: (state) => state.userId,
    getPrivateMessages: (state) => state.privateMessages,
    getOnlineFriends: (state) => state.onlineFriends,
    getMessageReceivedTick: (state) => state.messageReceivedTick,
  },
  mutations: {
    SET_WS(state, ws) {
      state.ws = ws;
    },
    SET_ONLINE_FRIENDS(state, onlineFriends) {
      state.onlineFriends = onlineFriends;
    },
    ADD_PRIVATE_MESSAGE(state, message) {
      state.privateMessages.push(message);
      state.messageReceivedTick += 1;
    },
    CLEAR_PRIVATE_MESSAGES(state) {
      state.privateMessages = [];
    },
    SET_USER_ID(state) {
      const profile = JSON.parse(localStorage.getItem('profile') || 'null');
      if (profile) {
        state.userId = profile?.result?._id ?? '';
      }
    },
    RESET_STATE(state) {
      Object.assign(state, initialState());
    },
  },
  actions: {
    async createChatConnection({ state, commit }) {
      try {
        commit('SET_USER_ID');
        const profile = JSON.parse(localStorage.getItem('profile') || 'null');
        const token = profile?.token;
        if (!token || state.ws) return;

        const baseUrl = process.env.VUE_APP_REALTIME_CHAT_URL;
        const ws = new WebSocket(`${baseUrl}${token}`);

        ws.onopen = () => {
          commit('SET_WS', ws);
        };

        ws.onmessage = (event) => {
          const data = JSON.parse(event.data);
          if (data.onlineFriends) {
            const uniqueFriends = Array.from(new Set(data.onlineFriends));
            commit('SET_ONLINE_FRIENDS', uniqueFriends);
          } else {
            commit('ADD_PRIVATE_MESSAGE', data);
          }
        };

        ws.onerror = (error) => {
          console.error('WebSocket error:', error);
        };

        ws.onclose = () => {
          commit('SET_WS', null);
        };
      } catch (error) {
        console.error('createChatConnection error:', error);
      }
    },
    async sendPrivateMessage({ state }, message) {
      if (state.ws && state.ws.readyState === WebSocket.OPEN) {
        state.ws.send(JSON.stringify(message));
      } else {
        console.warn('sendPrivateMessage: socket not open, message not sent');
      }
    },
    async stopConnectionToChat({ state, commit }) {
      try {
        if (state.ws) {
          state.ws.close();
          commit('SET_WS', null);
        }
      } catch (error) {
        console.error('stopConnectionToChat error:', error);
      }
    },
  },
};

export default RealTimeChat;