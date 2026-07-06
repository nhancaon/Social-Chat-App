export default function resetOnLogout(store) {
  store.subscribe((mutation) => {
    if (mutation.type === 'auth/LOGOUT') {
      Object.keys(store.state).forEach((moduleName) => {
        if (moduleName === 'auth') return

        const mutationType = `${moduleName}/RESET_STATE`

        // kiểm tra mutation có tồn tại trước khi commit,
        // tránh lỗi âm thầm nếu module nào đó quên khai báo RESET_STATE
        if (store._mutations[mutationType]) {
          store.commit(mutationType)
        } else if (process.env.NODE_ENV !== 'production') {
          console.warn(
            `[resetOnLogout] Module "${moduleName}" thiếu mutation RESET_STATE — state cũ có thể bị giữ lại sau logout.`
          )
        }
      })
    }
  })
}