import * as api from '../api/index.js';

const Chat = {
  state: {
    unReadedMsgsNUM: 0
  },
  getters: {
    getUnReadedMsg: (state) => () => {
      return state.unReadedMsgsNUM
    }
  },
  mutations: {
    updateUnreadedMsg(state, payload) {
      state.unReadedMsgsNUM = payload
    }
  },
  actions: {
    async GetUnreadedMessageNum(context, uid) {
      try {
        let { data } = await api.GetUnreadedMsgNum(uid)
        context.commit('updateUnreadedMsg', data.total)
        return data;
      } catch (error) {
        console.log(error)
      }
    },
    async GetChatMsgsBetweenTwoUsers(context, ndata) {
      try {
        let { data } = await api.GetMsgsBetweenTwoUsersByNum(ndata.from, ndata.firstuid, ndata.seconduid)
        return data;
      } catch (error) {
        console.log(error)
      }
    },
    async SendMessage(context, sdata) {
      try {
        const msg =
        {
          "content": sdata.content,
          "sender": sdata.sender,
          "recever": sdata.recever,
        }
        // console.log("cont", msg)
        let { data } = await api.SendMessage(msg)
        console.log('chat store sned message', data)

        return data;
      } catch (error) {
        console.log(error)
      }
    },
    async MarkMsgsAsReaded(context, datau) {
      try {
        let { data } = await api.markMsgAsReaded(datau.mainuid, datau.otheruid)
        var olunreaded = context.state.unReadedMsgsNUM;
        var unreaded = datau.GetunReadedmessage;

        var finalnum = olunreaded - unreaded;
        context.commit('updateUnreadedMsg', finalnum);

        // console.log()
        return data;

      } catch (error) {
        console.log(error)
      }
    }
  }
}

export default Chat;






