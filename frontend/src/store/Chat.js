import * as api from '../api/index.js';


function initialState() {
  return {
    unreadMsgCount: 0,
  }
}

const Chat = {
  namespaced: true,
  state: initialState(),
  getters: {
    getUnreadMsgCount: (state) => state.unreadMsgCount
  },
  mutations: {
    UPDATE_UNREAD_MSG_COUNT(state, payload) {
      state.unreadMsgCount = payload
    },
    RESET_STATE(state) {
      Object.assign(state, initialState())
    },
  },
  actions: {
    async getUnreadMessageNum({ commit }, uid) {
      try {
        const { data } = await api.getUnreadMessageCount(uid)
        commit('UPDATE_UNREAD_MSG_COUNT', data.total)
        return data
      } catch (error) {
        console.error('getUnreadMessageNum error:', error)
        throw error
      }
    },

    async getChatMsgsBetweenTwoUsers(_, payload) {
      try {
        const { data } = await api.getMessagesByPage(
          payload.from,
          payload.firstuid,
          payload.seconduid
        )
        return data
      } catch (error) {
        console.error('getChatMsgsBetweenTwoUsers error:', error)
        throw error
      }
    },

    async sendMessage(_, payload) {
      try {
        const msg = {
          content: payload.content,
          sender: payload.sender,
          receiver: payload.receiver,
        }
        const { data } = await api.sendMessage(msg)
        return data
      } catch (error) {
        console.error('sendMessage error:', error)
        throw error
      }
    },

    async markMsgsAsRead({ dispatch }, payload) {
      try {
        const { data } = await api.markMessageAsRead(payload.mainuid, payload.otheruid)
        await dispatch('getUnreadMessageNum', payload.mainuid)
        return data
      } catch (error) {
        console.error('markMsgsAsRead error:', error)
        throw error
      }
    }
  }
}

export default Chat;