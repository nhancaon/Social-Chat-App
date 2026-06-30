import * as api from '../api/index.js'

const NotificationStore = {
  state: {
    unReadedNotification: 0
  },
  getters: {
    GetUnReadedNotification: (state) => () => {
      return state.unReadedNotification
    },
  },
  mutations: {
    updateUnReadedNofification(state, payload) {
      state.unReadedNotification = payload
    },
  },
  actions: {
    async GetUnReadedNotifyNum(context, id) {
      try {
        let { data } = await api.GetNofificationForUser(id)
        let numofunreadednot = 0;
        data.notifications.forEach(el => {
          if (!el.isreded) {
            numofunreadednot++;
          }
        });

        context.commit('updateUnReadedNofification', numofunreadednot)

        return data.notifications;
      } catch (error) {
        console.log(error)
      }
    },
    async MarkNotifyAsReaded(context, id) {
      try {

        let { data } = await api.MartNotificationAsReaded(id)
        context.commit('updateUnReadedNofification', 0)

        return data;

      } catch (error) {
        console.log(error)
      }
    }
  }
}



export default NotificationStore;
