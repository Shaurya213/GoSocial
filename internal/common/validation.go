package common

import(
	"errors"
	"regexp"
	"strings"
)


var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)

func ValidateHandle(handle string) error {
	handle = strings.TrimSpace(handle)
	if len(handle) < 3 || len(handle) > 50 {
		return errors.New("Handle must be between 3 and 50 characters")
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(handle) {
		return errors.New("handle can only contain letters, numbers, and undrscores")
	}

	return nil
}

func ValidatePassword(password string) error{
	if len(password) < 6{
		return errors.New("password must be atleast 6 characters long")

	}

	if len(password) > 100 {
		return errors.New("password is too long baby!")
	}

	return nil
}


func ValidateEmail(email string) error {
	if email == "" {
		return nil
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}