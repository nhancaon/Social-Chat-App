package validation

import (
	"Server/models"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var ValidatorUser = validator.New()

// authInput is a dedicated parse/validate target for signup & signin bodies —
// kept separate from models.UserModel so the request's plaintext "password"
// field is always readable here, independent of how UserModel tags its own
// Password field for storage/response purposes.
type authInput struct {
	Email    string `validate:"required"`
	Password string `validate:"required,min=5"`
}

func ValidateUser(c *fiber.Ctx) error {
	var errors []*models.IError
	var body authInput

	if err := c.BodyParser(&body); err != nil {
		return err
	}

	err := ValidatorUser.Struct(body)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var el models.IError
			el.Field = err.Field()
			el.Tag = err.Tag()
			errors = append(errors, &el)
		}
		return c.Status(fiber.StatusBadRequest).JSON(errors)
	}
	// ok
	return c.Next()
}
