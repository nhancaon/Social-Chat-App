package validation

import (
	"Server/models"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var ValidatorPost = validator.New()

func ValidatePost(c *fiber.Ctx) error {
	var errors []*models.IError
	var body models.CreateOrUpdatePost

	if err := c.BodyParser(&body); err != nil {
		return err
	}

	err := ValidatorPost.Struct(body)
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
