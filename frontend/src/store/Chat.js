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
    updateUnreadMsgCount(state, payload) {
      state.unreadMsgCount = payload
    },
    RESET_STATE(state) {
      Object.assign(state, initialState())
    },
  },
  actions: {
    async GetUnreadMessageNum({ commit }, uid) {
      try {
        const { data } = await api.GetUnreadedMsgNum(uid)
        commit('updateUnreadMsgCount', data.total)
        return data
      } catch (error) {
        console.error('GetUnreadMessageNum error:', error)
        throw error
      }
    },

    async GetChatMsgsBetweenTwoUsers(_, payload) {
      try {
        const { data } = await api.GetMsgsBetweenTwoUsersByNum(
          payload.from,
          payload.firstuid,
          payload.seconduid
        )
        return data
      } catch (error) {
        console.error('GetChatMsgsBetweenTwoUsers error:', error)
        throw error
      }
    },

    async SendMessage(_, payload) {
      try {
        const msg = {
          content: payload.content,
          sender: payload.sender,
          receiver: payload.receiver,
        }
        const { data } = await api.SendMessage(msg)
        return data
      } catch (error) {
        console.error('SendMessage error:', error)
        throw error
      }
    },

    async MarkMsgsAsReaded({ dispatch }, payload) {
      try {
        const { data } = await api.markMsgAsReaded(payload.mainuid, payload.otheruid)
        await dispatch('GetUnreadMessageNum', payload.mainuid)
        return data
      } catch (error) {
        console.error('MarkMsgsAsReaded error:', error)
        throw error
      }
    }
  }
}

export default Chat;