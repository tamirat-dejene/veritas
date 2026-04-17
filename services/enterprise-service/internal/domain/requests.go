package domain

// CreateUserRequest is the payload for creating an enterprise user.
type CreateUserRequest struct {
	Email     string  `json:"email"     binding:"required,email"`
	Password  string  `json:"password"  binding:"required,min=8"`
	Role      Role    `json:"role"      binding:"required"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Phone     *string `json:"phone"`
	Honorific *string `json:"honorific"`
}

// UpdateUserRequest is the payload for updating an enterprise user.
type UpdateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Phone     *string `json:"phone"`
	Honorific *string `json:"honorific"`
	Role      *Role   `json:"role"`
}

// DomainValidationResult is the response from a custom domain validation check.
type DomainValidationResult struct {
	Domain         string `json:"domain"`
	TXTRecordFound bool   `json:"txt_record_found"`
	CNAMEFound     bool   `json:"cname_found"`
	Valid          bool   `json:"valid"`
	Details        string `json:"details"`
}

// ChangePasswordRequest is the payload for a user changing their own password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password"     binding:"required,min=8"`
}
