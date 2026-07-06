<template>
    <q-page class="constrain q-pa-md">
        <div class="row q-col-gutter-lg">
            <div class="col-12">

                <q-list bordered padding>

                    <div v-for="notify in NotifyList" :key="notify._id">
                        <q-item clickable @click="MoveToThePath(notify)" :class="{ 'text-red': !notify.isRead }">
                            <q-item-section top avatar>
                                <q-avatar v-if="notify?.user?.avatar">
                                    <img :src="notify?.user?.avatar">
                                </q-avatar>
                                <q-avatar v-else>
                                    <img src="https://cdn-icons-png.flaticon.com/512/3237/3237472.png">
                                </q-avatar>

                            </q-item-section>

                            <q-item-section>
                                <q-item-label>{{ notify?.details }}</q-item-label>
                                <q-item-label>{{ notify?.user?.name }}</q-item-label>
                            </q-item-section>
                        </q-item>
                        <q-separator spaced />
                    </div>


                </q-list>

            </div>
        </div>
    </q-page>


</template>

<script>
import { mapGetters, mapActions } from 'vuex';
// import {watch} from 'vue';

export default {
    name: 'Notification-Component',
    data() {
        return {
            NotifyList: []
        }
    },
    async mounted() {
        var id = this.GetAuthData.result._id;
        this.NotifyList = await this.GetUnReadNotifyNum(id)
        console.log("notifilist", this.NotifyList)
        // mark notification as readed
        setTimeout(() => {
            this.NotifyList.forEach(async el => {
                if (!el.isRead) {
                    await this.MarkAllNotifyAsReaded(id);
                    el.isRead = true;
                }
            })
        }, 500);
    },
    computed: {
        ...mapGetters('auth', ['GetAuthData'])
    },
    methods: {
        ...mapActions('notification', ['GetUnReadNotifyNum', 'MarkAllNotifyAsReaded']),

        MoveToThePath(notify) {
            if (notify?.details.toString().includes("Post")) {
                this.$router.push(`/PostDetail/${notify.targetid}`);
            } else {
                this.$router.push(`/Profile/${notify.targetid}`);
            }
        }
    }

}

</script>