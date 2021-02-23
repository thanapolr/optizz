package optizz

import (
	validator "gopkg.in/go-playground/validator.v9"
	"sync"
)

var (
	validatorObj  *validator.Validate
	validatorOnce sync.Once
)

// RegisterValidation registers a custom validation on the validator.Validate instance of the package
// NOTE: calling this function may instantiate the validator itself.
// NOTE: this function is not thread safe, since the validator validation registration isn't
func RegisterValidation(tagName string, validationFunc validator.Func) error {
	initValidator()
	return validatorObj.RegisterValidation(tagName, validationFunc)
}

func initValidator() {
	validatorOnce.Do(func() {
		validatorObj = validator.New()
		validatorObj.SetTagName(ValidationTag)
	})
}
