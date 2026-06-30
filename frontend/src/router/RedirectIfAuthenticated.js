import store from "@/store";

export default function RedirectIfAuthenticated(to, from, next) {
  if (store.state.Auth.authData) {
    next('/');
  } else {
    next();
  }
}