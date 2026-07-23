<template>
    <q-page class="constrain q-pa-md">
        <q-list bordered padding>
            <div v-for="notify in notifications" :key="notify._id">
                <q-item clickable @click="MoveToThePath(notify)" :class="{ 'text-red': !notify.isRead }">
                    <q-item-section top avatar>
                        <q-avatar v-if="notify?.user?.avatar">
                            <img :src="notify.user.avatar">
                        </q-avatar>
                        <q-avatar v-else>
                            <img src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png">
                        </q-avatar>
                    </q-item-section>
                    <q-item-section>
                        <q-item-label>{{ notify.details }}</q-item-label>
                        <q-item-label caption>{{ notify.user?.name }}</q-item-label>
                    </q-item-section>
                </q-item>
                <q-separator spaced />
            </div>
        </q-list>
    </q-page>
</template>

<script>
import { mapGetters, mapActions } from 'vuex';

export default {
    name: 'NotificationComponent',
    computed: {
        ...mapGetters('auth', ['currentUserId']),
        ...mapGetters('notification', ['getNotifications']),
        notifications() {
            return this.getNotifications;
        }
    },
    methods: {
        ...mapActions('notification', ['GetUnReadNotifyNum', 'MarkAllNotifyAsReaded']),
        MoveToThePath(notify) {
            if (notify.details?.toString().includes('Post')) {
                this.$router.push(`/PostDetail/${notify.targetid}`);
            } else {
                this.$router.push(`/Profile/${notify.targetid}`);
            }
        }
    },
    async mounted() {
        const userId = this.currentUserId;
        if (userId) {
            await this.GetUnReadNotifyNum(userId);
            await this.MarkAllNotifyAsReaded(userId);
        }
    }
}
</script>