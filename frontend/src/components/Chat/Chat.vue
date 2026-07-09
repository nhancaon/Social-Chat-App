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
                                    <q-badge color="positive" rounded />
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
                    <div class="q-pa-md row justify-center" style=" overflow-y: auto; max-height: 400px;"
                        ref="messageContainer" @scroll="handleScroll">
                        <div v-for="msg in messageBetweenUsers" :key="msg._id" style="width: 100%;">
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
            uniqueOnlineUsers: [],
        };
    },
    computed: {
        ...mapGetters('auth', ['GetAuthData'])
    },
    async mounted() {
        this.MainUserData = this.GetAuthData.result;
        this.GetUsList();
    },
    methods: {
        ...mapActions('chat', [
            'GetUnreadMessageNum',
            'GetChatMsgsBetweenTwoUsers',
            'SendMessage',
            'MarkMsgsAsReaded',
            'GetUserFollowersFollowing'
        ]),
        ...mapActions('users', ['GetUserFollowersFollowing']),
        handleScroll() {
            const container = this.$refs.messageContainer;
            if (container.scrollTop === 0) {
                // scorelled to the top
                this.GetOldestMessgesBetweenUsers();
            }
        },

        async GetOldestMessgesBetweenUsers() {
            this.messagelistnum = this.messagelistnum + 1;
            var firstuid = this.MainUserData._id
            var seconduid = this.selectedUser._id
            var from = this.messagelistnum;
            var ndata = { from, firstuid, seconduid };

            var { msgs } = await this.GetChatMsgsBetweenTwoUsers(ndata);
            this.messageBetweenUsers.unshift(...msgs);

        },
        scrollDownFunction() {
            const container = this.$refs.messageContainer;
            container.scrollTop = container.scrollHeight;
        },
        async CallMarkMsgAsReaded(user) {
            var mainuid = this.MainUserData._id;
            var otheruid = user._id;
            var GetunReadedmessage = 0

            this.contacts.forEach(
                user => {
                    if (String(otheruid) == String(user._id)) {
                        GetunReadedmessage = user.unReadedmessage
                    }
                }
            )

            var data = { mainuid, otheruid, GetunReadedmessage }
            var { isMarked } = await this.MarkMsgsAsReaded(data);

            if (isMarked) {
                this.contacts.forEach(user => {
                    if (String(otheruid) == String(user._id)) {
                        user.unReadedmessage = 0;
                    }
                })
            }
        },
        async GetUnreadedMsgList() {
            var { messages } = await this.GetUnreadMessageNum(this.MainUserData._id);
            this.contacts.forEach(user => {
                messages.forEach(msg => {
                    if (String(msg.otherUserid) == String(user._id)) {
                        user.unReadedmessage = Number(msg.numOfUnreadMessages);
                    }
                })
            })
        },
        async GetUsList() {
            this.contacts = [];
            var glist = await this.GetUserFollowersFollowing();
            this.contacts = glist;
            if (this.contacts) {
                this.GetUnreadedMsgList();
            }
        },
        async selectUser(user) {
            this.selectedUser = null;
            this.messageBetweenUsers = [];

            this.selectedUser = user;
            this.messagelistnum = 0;
            var firstuid = this.MainUserData._id;
            var seconduid = user._id;
            var from = 0;
            var ndata = { from, firstuid, seconduid };
            var { msgs } = await this.GetChatMsgsBetweenTwoUsers(ndata);
            this.messageBetweenUsers.push(...msgs);
            setTimeout(() => {
                this.scrollDownFunction();
                this.CallMarkMsgAsReaded(user)
            }, 100);

        },
        Sendmessage() {
            var content = this.messageToSend.text;
            var sender = this.MainUserData._id;
            var receiver = this.selectedUser._id;

            var sdata = { content, sender, receiver };

            var sucess = this.SendMessage(sdata);
            if (sucess) {
                this.messageBetweenUsers.push(sdata);
                setTimeout(() => {
                    this.scrollDownFunction();
                }, 100);
            }
            this.messageToSend.text = '';
        }

    }
}

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
