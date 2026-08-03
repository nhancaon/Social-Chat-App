<template>
  <q-page class="constrain q-pa-md auth-page">
    <!-- tab switcher, mobile only -->
    <div class="lt-sm q-mb-md">
      <q-tabs v-model="activeTab" dense class="text-grey" active-color="primary" indicator-color="primary"
        align="justify" narrow-indicator>
        <q-tab name="signin" label="Sign in" />
        <q-tab name="signup" label="Sign up" />
      </q-tabs>
      <q-separator />
    </div>

    <div class="row q-col-gutter-lg justify-center">
      <div class="col-12 col-sm-5" v-show="$q.screen.gt.xs || activeTab === 'signin'">
        <q-card class="auth-card">
          <h1 class="text-h6 text-center">Sign in</h1>
          <q-card-section>
            <form @submit.prevent.stop="Login" class="q-gutter-md">
              <q-input filled v-model="signinForm.email" label="Your Email *" hint="Email" lazy-rules />
              <q-input filled v-model="signinForm.password" label="Your Password *" hint="Password" type="password"
                lazy-rules />
              <div>
                <q-btn label="Sign in" type="submit" color="primary" class="full-width" :loading="signinLoading"
                  :disable="signinLoading" rounded />
              </div>
            </form>
          </q-card-section>
        </q-card>
      </div>
      <div class="col-12 col-sm-7" v-show="$q.screen.gt.xs || activeTab === 'signup'">
        <q-card class="auth-card">
          <h1 class="text-h6 text-center">Sign up | Create New Account</h1>
          <q-card-section>
            <form @submit.prevent.stop="Register" class="q-gutter-md">
              <div class="row q-col-gutter-sm">
                <div class="col-12 col-sm-6">
                  <q-input filled v-model="signupForm.firstName" label="Your First Name *" hint="First name"
                    lazy-rules />
                </div>
                <div class="col-12 col-sm-6">
                  <q-input filled v-model="signupForm.lastName" label="Your Last Name *" hint="Last name"
                    lazy-rules />
                </div>
              </div>
              <q-input filled v-model="signupForm.email" label="Your Email *" hint="Email" lazy-rules />
              <q-input filled v-model="signupForm.password" type="password" label="Your Password *" hint="Password"
                lazy-rules />
              <div>
                <q-btn label="Create New Account" type="submit" color="positive" class="full-width"
                  :loading="signupLoading" :disable="signupLoading" rounded />
              </div>
            </form>
          </q-card-section>
        </q-card>
      </div>
    </div>
  </q-page>
</template>

<script>
import { mapActions } from 'vuex'

export default {
  name: 'AuthView',
  data() {
    return {
      activeTab: 'signin',
      signinLoading: false,
      signupLoading: false,
      signinForm: {
        email: '',
        password: '',
      },
      signupForm: {
        email: '',
        password: '',
        firstName: '',
        lastName: '',
      },
    }
  },
  methods: {
    ...mapActions('auth', ['signin', 'signup']),
    ...mapActions('realTimeNotify', ['connectToNotifications']),
    ...mapActions('realTimeChat', ['createChatConnection']),
    notifyError(message) {
      this.$q.notify({
        icon: 'eva-alert-circle-outline',
        type: 'negative',
        message,
      })
    },
    notifySuccess(message) {
      this.$q.notify({
        icon: 'eva-alert-circle-outline',
        type: 'positive',
        message,
      })
    },

    validateSignin() {
      if (!this.signinForm.email) {
        this.notifyError('Email is required')
        return false
      }
      if (!this.signinForm.password) {
        this.notifyError('Password is required')
        return false
      }
      return true
    },

    validateSignup() {
      for (const key in this.signupForm) {
        if (!this.signupForm[key]) {
          this.notifyError(`${key} is required`)
          return false
        }
      }
      return true
    },

    async Login() {
      if (!this.validateSignin()) return

      this.signinLoading = true
      try {
        await this.signin({
          email: this.signinForm.email,
          password: this.signinForm.password,
        })
        this.notifySuccess('Successfully signed in')
        this.connectToNotifications()
        this.createChatConnection()
        this.$router.push('/')
      } catch (error) {
        const message =
          error?.response?.data?.error || error?.response?.data?.message || 'Sign in failed'
        this.notifyError(`Error: ${message}`)
      } finally {
        this.signinLoading = false
      }
    },

    async Register() {
      if (!this.validateSignup()) return

      this.signupLoading = true
      try {
        await this.signup(this.signupForm)
        this.notifySuccess('Successfully signed up')
        this.connectToNotifications()
        this.createChatConnection()
        this.$router.push('/')
      } catch (error) {
        const message = error?.response?.data?.message || 'Sign up failed'
        this.notifyError(`Error: ${message}`)
      } finally {
        this.signupLoading = false
      }
    },
  },
}
</script>

<style lang="sass" scoped>
.auth-card
  border-radius: 12px
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08)
  padding: 10px
  width: 100%

@media (max-width: 599px)
  .auth-page
    padding: 12px 8px
</style>