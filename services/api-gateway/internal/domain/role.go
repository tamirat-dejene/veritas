package domain

type Role string

const (
	RoleAll             Role = "All"
	RoleSystemAdmin     Role = "SystemAdmin"
	RoleEnterpriseAdmin Role = "EnterpriseAdmin"
	RoleEnterpriseAuto  Role = "EnterpriseAuto"
	RoleEnterpriseStaff Role = "EnterpriseStaff"
	RoleExamCandidate   Role = "ExamCandidate"
)
