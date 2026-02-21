package domain

type Role string

const (
	RoleAll             Role = "All"
	RoleSuperAdmin      Role = "SuperAdmin"
	RoleEnterpriseAdmin Role = "EnterpriseAdmin"
	RoleStaff           Role = "Staff"
	RoleCandidate       Role = "Candidate"
	RoleSystem          Role = "System"
	RoleEnterpriseStaff Role = "EnterpriseStaff"
	RoleAdmin           Role = "Admin"
)
