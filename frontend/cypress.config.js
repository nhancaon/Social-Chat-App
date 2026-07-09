const { defineConfig } = require("cypress");

module.exports = defineConfig({
  allowCypressEnv: false,

  e2e: {
    baseUrl: 'http://localhost:80',
    setupNodeEvents(on, config) {
      // implement node event listeners here
    },
  },
});
