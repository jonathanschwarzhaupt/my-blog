package validator

import (
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestNotBlank(t *testing.T) {
	assert.False(t, NotBlank(""))
	assert.False(t, NotBlank("   "))
	assert.True(t, NotBlank("hello"))
}

func TestMaxChars(t *testing.T) {
	assert.True(t, MaxChars("hello", 5))
	assert.False(t, MaxChars("hello!", 5))
}

func TestMinChars(t *testing.T) {
	assert.True(t, MinChars("hello", 5))
	assert.False(t, MinChars("hi", 5))
}

func TestPermittedValue(t *testing.T) {
	assert.True(t, PermittedValue("go", "go", "rust"))
	assert.False(t, PermittedValue("python", "go", "rust"))
}

func TestValidator_CheckField(t *testing.T) {
	v := &Validator{}
	v.CheckField(NotBlank(""), "so_what", "cannot be blank")

	assert.False(t, v.Valid())
	assert.Equal(t, v.FieldErrors["so_what"], "cannot be blank")
}

func TestValidator_Valid_NoErrors(t *testing.T) {
	v := &Validator{}
	v.CheckField(NotBlank("something"), "so_what", "cannot be blank")

	assert.True(t, v.Valid())
}
