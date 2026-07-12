package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEmail_Valid(t *testing.T) {
	v := New()
	v.ValidateEmail("test@example.com")
	assert.False(t, v.HasErrors())
}

func TestValidateEmail_Invalid(t *testing.T) {
	v := New()
	v.ValidateEmail("invalid-email")
	assert.True(t, v.HasErrors())
	assert.Equal(t, "email", v.Errors[0].Field)
}

func TestValidatePassword_Valid(t *testing.T) {
	v := New()
	v.ValidatePassword("SecurePass123!")
	assert.False(t, v.HasErrors())
}

func TestValidatePassword_TooShort(t *testing.T) {
	v := New()
	v.ValidatePassword("Ab1!")
	assert.True(t, v.HasErrors())
}

func TestValidatePassword_NoUpper(t *testing.T) {
	v := New()
	v.ValidatePassword("securepass123!")
	assert.True(t, v.HasErrors())
}

func TestValidatePassword_NoLower(t *testing.T) {
	v := New()
	v.ValidatePassword("SECUREPASS123!")
	assert.True(t, v.HasErrors())
}

func TestValidatePassword_NoDigit(t *testing.T) {
	v := New()
	v.ValidatePassword("SecurePass!")
	assert.True(t, v.HasErrors())
}

func TestValidatePassword_NoSpecial(t *testing.T) {
	v := New()
	v.ValidatePassword("SecurePass123")
	assert.True(t, v.HasErrors())
}

func TestValidateName_Valid(t *testing.T) {
	v := New()
	v.ValidateName("John Doe")
	assert.False(t, v.HasErrors())
}

func TestValidateName_TooShort(t *testing.T) {
	v := New()
	v.ValidateName("J")
	assert.True(t, v.HasErrors())
}

func TestValidateRole_Valid(t *testing.T) {
	v := New()
	v.ValidateRole("admin")
	assert.False(t, v.HasErrors())
}

func TestValidateRole_Invalid(t *testing.T) {
	v := New()
	v.ValidateRole("superadmin")
	assert.True(t, v.HasErrors())
}

func TestMultipleErrors(t *testing.T) {
	v := New()
	v.ValidateEmail("invalid")
	v.ValidatePassword("weak")
	assert.True(t, v.HasErrors())
	assert.Len(t, v.Errors, 2)
}