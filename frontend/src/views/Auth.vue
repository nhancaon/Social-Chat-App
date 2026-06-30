<template>
 <q-page class="constrain q-pa-md">
    <div class="row q-col-gutter-lg">
        <div class="col-5">
            <q-card class="my-card" style="width: 100%; padding: 10px;">
                <h1 class="text-h6 text-center">Sign in</h1>
                <q-card-section>
                    <form @submit.prevent.stop="Login" class="q-gutter-md">
                        <q-input
                          filled
                           v-model="signinForm.email"
                           label="Your Email *"
                           hint="Email"
                           lazy-rules
                        />
                        <q-input
                           filled
                           v-model="signinForm.password"
                           label="Your Password *"
                           hint="Password"
                           type="password"
                           lazy-rules
                        />
                        <div>
                            <q-btn
                              label="Sign in"
                              type="submit"
                              color="primary"
                              :loading="signinLoading"
                              :disable="signinLoading"
                            />
                        </div>
                    </form>
                </q-card-section>
            </q-card>
        </div>
        <div class="col-7">
            <q-card class="my-card" style="width: 100%; padding: 10px;">
                <h1 class="text-h6 text-center">Sign up | Create New Account</h1>
                <q-card-section>
                    <form @submit.prevent.stop="Register" class="q-gutter-md">
                        <q-input
                          filled
                           v-model="signupForm.firstName"
                           label="Your First Name *"
                           hint="First name"
                           lazy-rules
                        />
                        <q-input
                          filled
                           v-model="signupForm.lastName"
                           label="Your Last Name *"
                           hint="Last name"
                           lazy-rules
                        />
                        <q-input
                          filled
                           v-model="signupForm.email"
                           label="Your Email *"
                           hint="Email"
                           lazy-rules
                        />
                        <q-input
                           filled
                           v-model="signupForm.password"
                           type="password"
                           label="Your Password *"
                           hint="Password"
                           lazy-rules
                        />
                        <div>
                            <q-btn
                              label="Create New Account"
                              type="submit"
                              color="positive"
                              :loading="signupLoading"
                              :disable="signupLoading"
                            />
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
    ...mapActions(['signin', 'signup']),

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
        this.$router.push('/')
      } catch (error) {
        const message =
          error?.response?.data?.message || error?.response?.data || 'Sign in failed'
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