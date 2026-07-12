package validator

import (
	"net/mail"
	"strings"
	"unicode"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Validator struct {
	Errors []ValidationError
}

func New() *Validator {
	return &Validator{
		Errors: make([]ValidationError, 0),
	}
}

func (v *Validator) HasErrors() bool {
	return len(v.Errors) > 0
}

func (v *Validator) AddError(field, message string) {
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

func (v *Validator) ValidateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	if err != nil {
		v.AddError("email", "invalid email format")
		return false
	}
	return true
}

func (v *Validator) ValidatePassword(password string) bool {
	if len(password) < 8 {
		v.AddError("password", "password must be at least 8 characters")
		return false
	}
	if len(password) > 128 {
		v.AddError("password", "password must not exceed 128 characters")
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	if !hasUpper {
		v.AddError("password", "password must contain at least one uppercase letter")
		return false
	}
	if !hasLower {
		v.AddError("password", "password must contain at least one lowercase letter")
		return false
	}
	if !hasDigit {
		v.AddError("password", "password must contain at least one digit")
		return false
	}
	if !hasSpecial {
		v.AddError("password", "password must contain at least one special character")
		return false
	}

	return true
}

func (v *Validator) ValidateName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 2 {
		v.AddError("name", "name must be at least 2 characters")
		return false
	}
	if len(name) > 255 {
		v.AddError("name", "name must not exceed 255 characters")
		return false
	}
	return true
}

func (v *Validator) ValidateRole(role string) bool {
	validRoles := map[string]bool{
		"user":  true,
		"admin": true,
		"moderator": true,
	}
	if !validRoles[role] {
		v.AddError("role", "invalid role. Must be one of: user, admin, moderator")
		return false
	}
	return true
}