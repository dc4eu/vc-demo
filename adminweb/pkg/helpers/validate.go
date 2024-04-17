package helpers

import (
	"context"
	"reflect"
	"strings"
	"vcweb1/pkg/logger"
	"vcweb1/pkg/model"

	"github.com/go-playground/validator/v10"
)

// Check checks for validation error
func Check(ctx context.Context, cfg *model.Cfg, s any, log *logger.Log) error {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		return name
	})

	if err := validate.Struct(s); err != nil {
		return NewErrorFromError(err)
	}

	return nil
}
