import { createStore } from 'vuex'
import Auth from './Auth'
import Users from './Users'
import Posts from './Posts'
import Notification from './Notification'
import Chat from './Chat'
import RealTimeNotify from './RealTimeNotify'
import RealTimeChat from './RealTimeChat'
import resetOnLogout from './plugins/resetOnLogout'


export default createStore({
  modules: {
    auth: Auth,
    users: Users,
    posts: Posts,
    notification: Notification,
    chat: Chat,
    realTimeNotify: RealTimeNotify,
    realTimeChat: RealTimeChat,
  },
  plugins: [resetOnLogout],
})