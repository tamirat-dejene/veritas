package emailtemplate

import (
	_ "embed"
)

//go:embed welcome_staff_email.html
var WelcomeStaffEmail string
