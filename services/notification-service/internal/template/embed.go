package emailtemplate

import (
	_ "embed"
)

//go:embed welcome_staff_email.html
var WelcomeStaffEmail string

//go:embed password_reset_email.html
var PasswordResetEmail string
