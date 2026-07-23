<template>
    <q-page class="constrain q-pa-md">
        <div class="row q-col-gutter-lg">
            <div class="col-12 chat-container">
                <div class="user-list">
                    <div class="q-pa-md">
                        <q-toolbar class="bg-primary text-white shadow-1">
                            <q-toolbar-title>Following & Followers</q-toolbar-title>
                        </q-toolbar>
                        <q-list bordered>
                            <q-item @click="selectUser(contact)" v-for="contact in contacts" :key="contact._id"
                                class="q-my-sm" clickable v-ripple>
                                <q-item-section avatar>
                                    <q-avatar v-if="!contact.imageUrl" color="primary" text-color="white">
                                        {{ contact.name[0] }}
                                    </q-avatar>
                                    <q-avatar v-else>
                                        <img :src="contact?.imageUrl">
                                    </q-avatar>
                                </q-item-section>
                                <q-item-section>
                                    <q-item-label>{{ contact.name }}</q-item-label>
                                </q-item-section>

                                <q-item-section side>
                                    <!-- online indicator now driven by realtime store -->
                                    <q-badge :color="isOnline(contact._id) ? 'positive' : 'grey'" rounded />
                                </q-item-section>

                                <q-item-section side v-if="contact.unReadedmessage && contact.unReadedmessage > 0">
                                    <q-badge color="negative" rounded :label="contact?.unReadedmessage" />
                                </q-item-section>
                            </q-item>
                        </q-list>
                    </div>
                </div>

                <!-- chat box  -->
                <div class="chat-messages" v-if="selectedUser != null" style="background: white;">
                    <div class="q-pa-md"
                        style="display: flex; flex-direction: column; overflow-x: hidden; overflow-y: auto; max-height: 400px;"
                        ref="messageContainer" @scroll="handleScroll">
                        <div v-for="(msg, idx) in messageBetweenUsers" :key="msg._id || idx"
                            style="width: 100%; flex-shrink: 0;">
                            <q-chat-message
                                :name="msg.sender === MainUserData._id ? MainUserData.name : selectedUser.name"
                                :avatar="msg.sender === MainUserData._id ? MainUserData.imageUrl : selectedUser.imageUrl"
                                :text="[msg.content]" :sent="msg.sender === MainUserData._id ? true : false" />
                        </div>
                    </div>

                    <q-separator spaced />
                    <q-input outlined v-model="messageToSend.text" @keyup.enter="Sendmessage" label="write message..">
                        <q-btn v-if="messageToSend.text != ''" @click="Sendmessage" flat round color="primary"
                            icon="eva-arrow-right" />
                    </q-input>
                </div>
            </div>
        </div>
    </q-page>
</template>

<script>
import { mapGetters, mapActions } from 'vuex';

export default {
    name: 'ChatComponent',
    data() {
        return {
            messageToSend: { text: '' },
            contacts: [],
            messageBetweenUsers: [],
            messagelistnum: 0,
            selectedUser: null,
            MainUserData: {},
        };
    },
    computed: {
        ...mapGetters('auth', ['GetAuthData']),
        ...mapGetters('realTimeChat', ['getPrivateMessages', 'getOnlineFriends', 'getMessageReceivedTick']),
    },
    watch: {
        // Plain primitive counter -> Vue reliably fires this every single
        // time a message arrives, unlike a deep watch on a nested array.
        getMessageReceivedTick() {
            this.handleIncomingMessages();
        },
    },
    async mounted() {
        this.MainUserData = this.GetAuthData.result;
        // Connection lifecycle is now owned by NavBar (mounted on every
        // page), so the socket stays alive across navigation. This call is
        // a harmless no-op if already connected (the store guards against
        // opening a second connection).
        this.GetUsList();
    },
    // No beforeUnmount closing the connection anymore - leaving the Chat
    // page must not kill realtime updates for the rest of the app.
    methods: {
        ...mapActions('chat', [
            'GetUnreadMessageNum',
            'GetChatMsgsBetweenTwoUsers',
            'SendMessage',
            'MarkMsgsAsReaded',
        ]),
        ...mapActions('users', ['GetUserFollowersFollowing']),
        ...mapActions('realTimeChat', [
            'sendPrivateMessage',
        ]),

        isOnline(userId) {
            const online = this.getOnlineFriends || [];
            return online.some((id) => String(id) === String(userId));
        },

        handleScroll() {
            const container = this.$refs.messageContainer;
            if (container.scrollTop === 0) {
                this.GetOldestMessgesBetweenUsers();
            }
        },

        async GetOldestMessgesBetweenUsers() {
            this.messagelistnum = this.messagelistnum + 1;
            const firstuid = this.MainUserData._id;
            const seconduid = this.selectedUser._id;
            const from = this.messagelistnum;
            const { msgs } = await this.GetChatMsgsBetweenTwoUsers({ from, firstuid, seconduid });
            if (msgs && msgs.length) {
                const existingIds = new Set(this.messageBetweenUsers.map((m) => String(m._id)));
                const newOnes = msgs.filter((m) => !existingIds.has(String(m._id)));
                this.messageBetweenUsers.unshift(...newOnes);
            }
        },

        scrollDownFunction() {
            const container = this.$refs.messageContainer;
            if (container) container.scrollTop = container.scrollHeight;
        },

        // Adds a message only if it isn't already in the list (by _id,
        // falling back to sender+content+createdAt for optimistic messages
        // that don't have a server _id yet).
        addMessageIfNew(msg) {
            const isDuplicate = this.messageBetweenUsers.some((m) => {
                if (msg._id && m._id) return String(m._id) === String(msg._id);
                return (
                    String(m.sender) === String(msg.sender) &&
                    String(m.receiver) === String(msg.receiver) &&
                    m.content === msg.content &&
                    m.createdAt === msg.createdAt
                );
            });
            if (!isDuplicate) {
                this.messageBetweenUsers.push(msg);
                return true;
            }
            return false;
        },

        // Keeps fetching older pages while the container has no scrollbar
        // yet (content shorter than its max-height) and there are still
        // more messages on the server.
        async fillContainerIfNeeded() {
            await this.$nextTick();
            const container = this.$refs.messageContainer;
            if (!container) return;

            let guard = 0; // safety limit so a backend bug can't loop forever
            while (container.scrollHeight <= container.clientHeight && guard < 20) {
                const before = this.messageBetweenUsers.length;
                await this.GetOldestMessgesBetweenUsers();
                await this.$nextTick();
                guard += 1;
                // no more messages returned -> stop trying
                if (this.messageBetweenUsers.length === before) break;
            }
        },

        async CallMarkMsgAsReaded(user) {
            const mainuid = this.MainUserData._id;
            const otheruid = user._id;
            let unreadCount = 0;

            this.contacts.forEach((c) => {
                if (String(otheruid) === String(c._id)) unreadCount = c.unReadedmessage;
            });

            const { isMarked } = await this.MarkMsgsAsReaded({
                mainuid,
                otheruid,
                GetunReadedmessage: unreadCount,
            });

            if (isMarked) {
                this.contacts.forEach((c) => {
                    if (String(otheruid) === String(c._id)) c.unReadedmessage = 0;
                });
            }
        },

        async GetUnreadedMsgList() {
            const { messages } = await this.GetUnreadMessageNum(this.MainUserData._id);
            this.contacts.forEach((c) => {
                messages.forEach((msg) => {
                    if (String(msg.otherUserid) === String(c._id)) {
                        c.unReadedmessage = Number(msg.numOfUnreadMessages);
                    }
                });
            });
        },

        async GetUsList() {
            this.contacts = [];
            const glist = await this.GetUserFollowersFollowing();
            this.contacts = glist || [];
            if (this.contacts.length) {
                this.GetUnreadedMsgList();
            }
        },

        async selectUser(user) {
            this.selectedUser = null;
            this.messageBetweenUsers = [];

            this.selectedUser = user;
            this.messagelistnum = 0;

            const firstuid = this.MainUserData._id;
            const seconduid = user._id;
            const { msgs } = await this.GetChatMsgsBetweenTwoUsers({ from: 0, firstuid, seconduid });
            this.messageBetweenUsers.push(...(msgs || []));

            this.$nextTick(async () => {
                await this.fillContainerIfNeeded();
                this.scrollDownFunction();
                this.CallMarkMsgAsReaded(user);
            });
        },

        // Called whenever a new message arrives through the WebSocket store
        handleIncomingMessages() {
            const messages = this.getPrivateMessages;
            if (!messages || messages.length === 0) return;

            const lastMsg = messages[messages.length - 1];

            // We already show our own outgoing messages optimistically in
            // Sendmessage(). If the server echoes them back to us over the
            // socket, skip here - otherwise it shows up as a duplicate
            // bubble (dedupe-by-createdAt can miss it since the server may
            // assign its own createdAt/_id different from our local copy).
            if (String(lastMsg.sender) === String(this.MainUserData._id)) {
                return;
            }

            if (this.selectedUser) {
                const isCurrentConversation =
                    String(lastMsg.sender) === String(this.selectedUser._id) &&
                    String(lastMsg.receiver) === String(this.MainUserData._id);

                if (isCurrentConversation) {
                    const wasAdded = this.addMessageIfNew(lastMsg);
                    if (wasAdded) this.$nextTick(() => this.scrollDownFunction());
                    this.CallMarkMsgAsReaded(this.selectedUser);
                    return;
                }
            }
            // message belongs to another conversation -> just refresh unread badges
            this.GetUnreadedMsgList();
        },

        async Sendmessage() {
            const text = this.messageToSend.text.trim();
            if (!text || !this.selectedUser) return;

            const message = {
                content: text,
                sender: this.MainUserData._id,
                receiver: this.selectedUser._id,
                createdAt: new Date().toISOString(),
            };

            this.messageToSend.text = '';

            try {
                if (this.isOnline(this.selectedUser._id)) {
                    // Receiver has an open WS connection: the backend's
                    // SendToReceiver saves to DB AND relays in realtime in
                    // one step, so we only call this path - calling REST
                    // too would save the same message a second time.
                    await this.sendPrivateMessage(message);
                } else {
                    // Receiver offline: the backend's WS relay would
                    // silently drop the message without saving it (it
                    // returns early when there's no connection). REST is
                    // the only path that guarantees persistence here.
                    await this.SendMessage(message);
                }

                // show immediately in our own UI
                this.addMessageIfNew(message);
                this.$nextTick(() => this.scrollDownFunction());
            } catch (error) {
                console.error('Sendmessage error:', error);
            }
        },
    },
};
</script>

<style scoped>
.chat-container {
    display: flex;
}

.chat-messages {
    flex: 1;
    padding: 10px;
}
</style>