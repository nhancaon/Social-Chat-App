<template>
  <div class="q-pa-md">
    <q-item v-if="currentUser" :to="{ path: `/Profile/${currentUser._id}` }" clickable v-ripple>
      <q-item-section avatar>
        <q-icon color="primary" name="eva-person-outline" />
      </q-item-section>
      <q-item-section>Profile</q-item-section>
    </q-item>

    <q-item v-else to="/Auth" clickable v-ripple>
      <q-item-section avatar>
        <q-icon color="primary" name="eva-person-outline" />
      </q-item-section>
      <q-item-section>Authentication</q-item-section>
    </q-item>
  </div>
</template>

<script>
import { mapGetters } from 'vuex';

export default {
  name: 'SideBar',
  computed: {
    ...mapGetters('auth', ['GetAuthData']),
    currentUser() {
      return this.GetAuthData?.result ?? null
    }
  }
}
</script>