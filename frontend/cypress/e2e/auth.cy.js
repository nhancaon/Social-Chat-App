describe("Auth Component Tests", () => {
  beforeEach(() => {
    cy.viewport(1200, 800) // > sm breakpoint: both cards shown side by side, tab switcher hidden
    cy.visit('/Auth')
  })

  describe('Page Layout', () => {
    it('should display signin and signup cards', () => {
      cy.get('.q-card').should('have.length', 2)

      cy.get('.col-sm-5').contains('Sign in').should('be.visible')
      cy.get('.col-sm-7').contains('Sign up | Create New Account').should('be.visible')
    })

    it('should have proper layout structure', () => {
      cy.get('.col-sm-5').should('exist')
      cy.get('.col-sm-7').should('exist')
      cy.get('.row').should('exist')
    })
  })

  describe('Signin Form', () => {
    it('should display all signin form elements', () => {
      cy.get('.col-sm-5 input').should('have.length', 2)

      cy.get('.col-sm-5').contains('Sign in').should('be.visible')
      cy.get('.col-sm-5 button[type="submit"]').should('be.visible')
    })

    it('should allow typing in signin inputs', () => {
      cy.get('.col-sm-5 input').eq(0)
        .type('test@example.com')
        .should('have.value', 'test@example.com')

      cy.get('.col-sm-5 input').eq(1)
        .type('password123')
        .should('have.value', 'password123')
    })

    it('should have password input type', () => {
      cy.get('.col-sm-5 input').eq(1)
        .should('have.attr', 'type', 'password')
    })
  })

  describe('Signup Form', () => {
    it('should display all signup form elements', () => {
      cy.get('.col-sm-7 input').should('have.length', 4)
      cy.contains('Your First Name *').should('be.visible')
      cy.contains('Your Last Name *').should('be.visible')
      cy.contains('Your Email *').should('be.visible')
      cy.contains('Your Password *').should('be.visible')

      cy.contains('Create New Account').should('be.visible')
    })

    it('should allow typing in all signup inputs', () => {
      // Thứ tự input trong template: firstName, lastName, email, password
      cy.get('.col-sm-7 input').eq(0)
        .type('John')
        .should('have.value', 'John')

      cy.get('.col-sm-7 input').eq(1)
        .type('Doe')
        .should('have.value', 'Doe')

      cy.get('.col-sm-7 input').eq(2)
        .type('j@example.com')
        .should('have.value', 'j@example.com')

      cy.get('.col-sm-7 input').eq(3)
        .type('password123')
        .should('have.value', 'password123')
    })
  })

  describe('Button Styling', () => {
    it('should have correct button colors', () => {
      cy.get('.col-sm-5 .q-btn').should('have.class', 'bg-primary')   // color="primary"
      cy.get('.col-sm-7 .q-btn').should('have.class', 'bg-positive')  // color="positive"
    })
  })

  describe('Form Interactions', () => {
    it('should show error for empty signin form (stops at first missing field)', () => {
      // validateSignin() check email trước -> chỉ hiện đúng 1 thông báo này
      cy.get('.col-sm-5 button[type="submit"]').click()
      cy.get('.q-notification').should('be.visible').and('contain', 'Email is required')
    })

    it('should show error for empty signup form (stops at first missing field)', () => {
      // validateSignup() lặp theo thứ tự khai báo trong data(): email, password, firstName, lastName
      // -> field "email" trống đầu tiên nên chỉ hiện đúng 1 thông báo "email is required"
      cy.get('.col-sm-7 button[type="submit"]').click()
      cy.get('.q-notification').should('be.visible').and('contain', 'email is required')
    })

    it('should show next required error after filling previous fields (signup)', () => {
      // Điền email -> submit -> validate dừng ở "password"
      cy.get('.col-sm-7 input').eq(2).type('j@example.com')
      cy.get('.col-sm-7 button[type="submit"]').click()
      cy.get('.q-notification').should('be.visible').and('contain', 'password is required')
    })
  })

  describe('Responsive Design', () => {
    it('should maintain layout on different screen sizes', () => {
      cy.viewport(375, 667)
      // < sm breakpoint: tab switcher active, only the "signin" tab's card is v-show'd
      cy.get('.q-card').should('have.length', 1)

      cy.viewport(768, 1024)
      cy.get('.col-sm-5').should('be.visible')
      cy.get('.col-sm-7').should('be.visible')

      cy.viewport(1200, 800)
      cy.get('.row').should('be.visible')
    })
  })
})
