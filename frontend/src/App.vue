<template>
  <q-layout view="lHh Lpr lFf">
    <NavBar/>
    <q-page-container class="bg-grey-1">
      <router-view/>

    </q-page-container>
  </q-layout>
</template>

<script>
import NavBar from '@/views/NavBar.vue'
import { mapActions } from 'vuex'

export default {
  name: 'MainLayout',
  methods: {
    ...mapActions(['initAuth']),
  },
  async mounted() {
    await this.initAuth()
  },
  components: { NavBar },
}
</script>

<style lang="scss">
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}

nav {
  padding: 30px;

  a {
    font-weight: bold;
    color: #2c3e50;

    &.router-link-exact-active {
      color: #42b983;
    }
  }
}
</style>