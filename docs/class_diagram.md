# Veritas — System Class Diagram

> System-level class diagram covering all five implemented Go microservices.  
> Interfaces (ports) are shown with `<<interface>>` stereotype; enumerations with `<<enumeration>>`.  
> Cross-service identity references (e.g. `enterprise_id`, `candidate_id`) are shown as dependency arrows rather than composition, since they cross service boundaries.

---

## Auth Service

```mermaid
classDiagram
    direction TB

    class User {
        +UUID ID
        +string Email
        +string PasswordHash
        +string Honorific
        +string FirstName
        +string LastName
        +string Phone
        +Role Role
        +UUID EnterpriseID
        +bool IsActive
        +bool IsDeleted
        +bool EmailVerified
        +time EmailVerifiedAt
        +int FailedLoginAttempts
        +time LockedUntil
        +time PasswordChangedAt
        +bool MustChangePassword
        +time LastLoginAt
        +string LastLoginIP
        +string LastUserAgent
        +time CreatedAt
        +time UpdatedAt
    }

    class RefreshToken {
        +UUID ID
        +UUID UserID
        +string TokenHash
        +time ExpiresAt
        +bool Revoked
        +time CreatedAt
    }

    class Role {
        <<enumeration>>
        SystemAdmin
        EnterpriseAdmin
        EnterpriseAuto
        EnterpriseStaff
        ExamCandidate
    }

    class UserRepository {
        <<interface>>
        +FindByEmail(ctx, email) User
        +FindByID(ctx, id) User
        +UpdateLoginSuccess(ctx, userID, ip, userAgent) error
        +UpdateLoginFailure(ctx, userID, lockUntil) error
    }

    class RefreshTokenRepository {
        <<interface>>
        +Create(ctx, token) error
        +FindByHash(ctx, tokenHash) RefreshToken
        +Revoke(ctx, tokenID) error
        +DeleteExpiredByUserID(ctx, userID, before) error
    }

    class TokenService {
        <<interface>>
        +GenerateAccessToken(user) string
        +GenerateRefreshToken() (raw, hash)
    }

    User --> Role : has
    RefreshToken --> User : belongs to
    UserRepository ..> User : manages
    RefreshTokenRepository ..> RefreshToken : manages
    TokenService ..> User : uses
```

---

## Enterprise Service

```mermaid
classDiagram
    direction TB

    class Enterprise {
        +UUID ID
        +string Slug
        +string DisplayName
        +string LegalName
        +string ContactEmail
        +UUID OwnerAccountID
        +EnterpriseStatus Status
        +time ApprovedAt
        +time SuspendedAt
        +time DeletedAt
        +UUID SubscriptionPlanID
        +SubscriptionStatus SubscriptionStatus
        +time CurrentPeriodStart
        +time CurrentPeriodEnd
        +string LogoURL
        +string PrimaryColor
        +string SecondaryColor
        +string CustomDomain
        +string ContactPhone
        +string City
        +string Country
        +map Settings
        +time RetentionUntil
        +time CreatedAt
        +time UpdatedAt
        +UUID CreatedBy
        +UUID UpdatedBy
    }

    class EnterpriseStatus {
        <<enumeration>>
        PendingApproval
        Active
        Suspended
        Deleted
    }

    class SubscriptionStatus {
        <<enumeration>>
        Trial
        Active
        PastDue
        Canceled
        Expired
    }

    class AuditLog {
        +UUID ID
        +UUID EnterpriseID
        +UUID ActorID
        +string ActorRole
        +AuditEvent Event
        +map Metadata
        +time CreatedAt
    }

    class AuditEvent {
        <<enumeration>>
        enterprise.approved
        enterprise.suspended
        enterprise.deleted
        enterprise.reactivated
        enterprise.hard_deleted
        enterprise.branding_updated
        enterprise.settings_updated
        subscription.updated
        subscription.canceled
        subscription.renewed
        subscription.payment_suspended
        user.created
        user.updated
        user.deactivated
        user.password_reset
        enterprise.domain_validated
    }

    class EnterpriseUser {
        +UUID ID
        +string Email
        +string PasswordHash
        +string FirstName
        +string LastName
        +Role Role
        +UUID EnterpriseID
        +bool IsActive
        +bool IsDeleted
        +bool EmailVerified
        +time CreatedAt
        +time UpdatedAt
    }

    class EnterpriseRepository {
        <<interface>>
        +Create(ctx, enterprise) error
        +FindByID(ctx, id) Enterprise
        +FindBySlug(ctx, slug) Enterprise
        +Update(ctx, enterprise) error
        +Delete(ctx, id) error
        +ListPaginated(ctx, filter) List~Enterprise~
        +HardDelete(ctx, id) error
    }

    class UserRepository {
        <<interface>>
        +Create(ctx, user) error
        +FindByID(ctx, id) User
        +FindByEmail(ctx, email) User
        +Update(ctx, user) error
        +Delete(ctx, id) error
        +ListByEnterprise(ctx, enterpriseID) List~User~
        +CountByEnterprise(ctx, enterpriseID) int
    }

    class AuditRepository {
        <<interface>>
        +Create(ctx, log) error
        +ListByEnterprise(ctx, enterpriseID) List~AuditLog~
    }

    class EnterpriseUsecase {
        <<interface>>
        +RegisterEnterprise(ctx, enterprise, owner) Enterprise
        +ApproveEnterprise(ctx, id, adminID) error
        +SuspendEnterprise(ctx, id, adminID) error
        +DeleteEnterprise(ctx, id, adminID) error
        +GetEnterprise(ctx, id) Enterprise
        +ListEnterprises(ctx, filter) List~Enterprise~
        +UpdateBranding(ctx, id, req, adminID) error
        +UpdateSettings(ctx, id, patch, adminID) error
        +UpdateSubscription(ctx, id, req, adminID) error
        +CancelSubscription(ctx, id, adminID) error
        +RenewSubscription(ctx, id, adminID) error
        +GetEnterpriseStatus(ctx, id) EnterpriseStatusResponse
        +GetEnterpriseSummary(ctx, id) EnterpriseSummary
        +GetAuditLogs(ctx, id) List~AuditLog~
    }

    class UserUsecase {
        <<interface>>
        +CreateEnterpriseUser(ctx, enterpriseID, req, adminID) User
        +ListEnterpriseUsers(ctx, enterpriseID) List~User~
        +GetEnterpriseUser(ctx, enterpriseID, userID) User
        +UpdateEnterpriseUser(ctx, enterpriseID, userID, req, adminID) error
        +DeactivateEnterpriseUser(ctx, enterpriseID, userID, adminID) error
        +ResetUserPassword(ctx, enterpriseID, userID, adminID) string
    }

    Enterprise --> EnterpriseStatus : has
    Enterprise --> SubscriptionStatus : has
    AuditLog --> Enterprise : references
    AuditLog --> AuditEvent : categorized by
    EnterpriseUser --> Enterprise : belongs to
    EnterpriseRepository ..> Enterprise : manages
    UserRepository ..> EnterpriseUser : manages
    AuditRepository ..> AuditLog : manages
    EnterpriseUsecase ..> EnterpriseRepository : uses
    EnterpriseUsecase ..> AuditRepository : uses
    UserUsecase ..> UserRepository : uses
```

---

## Exam Service

```mermaid
classDiagram
    direction TB

    class Question {
        +UUID ID
        +UUID EnterpriseID
        +QuestionType Type
        +string Topic
        +DifficultyLevel Difficulty
        +string Title
        +string Content
        +string MediaURL
        +int Points
        +float64 NegativePoints
        +map Metadata
        +bool IsActive
        +UUID CreatedBy
        +time CreatedAt
        +time UpdatedAt
    }

    class QuestionOption {
        +UUID ID
        +UUID QuestionID
        +string Content
        +bool IsCorrect
    }

    class QuestionType {
        <<enumeration>>
        MCQ
        TrueFalse
        ShortAnswer
        Essay
    }

    class DifficultyLevel {
        <<enumeration>>
        Easy
        Medium
        Hard
    }

    class Exam {
        +UUID ID
        +UUID EnterpriseID
        +string Title
        +string Description
        +int DurationMinutes
        +float64 PassingScorePercent
        +bool NegativeMarking
        +int MaxParticipants
        +string InvitationMethod
        +ExamStatus Status
        +UUID TemplateSourceID
        +time ScheduledStart
        +time ScheduledEnd
        +map Settings
        +UUID CreatedBy
        +time CreatedAt
        +time UpdatedAt
    }

    class ExamStatus {
        <<enumeration>>
        Draft
        Scheduled
        Active
        Closed
        Archived
    }

    class ExamQuestion {
        +UUID ID
        +UUID ExamID
        +UUID QuestionID
        +int PointsOverride
        +int OrderIndex
    }

    class ExamRandomizationRule {
        +UUID ID
        +UUID ExamID
        +string Topic
        +DifficultyLevel Difficulty
        +int QuestionCount
    }

    class QuestionRepository {
        <<interface>>
        +Create(ctx, q) error
        +GetByID(ctx, id, enterpriseID) Question
        +ListByEnterprise(ctx, enterpriseID) List~Question~
        +Update(ctx, q) error
        +Delete(ctx, id, enterpriseID) error
    }

    class ExamRepository {
        <<interface>>
        +Create(ctx, exam) error
        +GetByID(ctx, id, enterpriseID) Exam
        +ListByEnterprise(ctx, enterpriseID) List~Exam~
        +Update(ctx, exam) error
        +Delete(ctx, id, enterpriseID) error
        +AddQuestion(ctx, examID, eq) error
        +RemoveQuestion(ctx, examID, questionID) error
        +AddRandomizationRule(ctx, examID, rule) error
        +UpdateRandomizationRule(ctx, examID, rule) error
        +DeleteRandomizationRule(ctx, examID, ruleID) error
    }

    class ExamUsecase {
        <<interface>>
        +CreateExam(ctx, exam, userID) Exam
        +GetExams(ctx, enterpriseID) List~Exam~
        +GetExam(ctx, id, enterpriseID) Exam
        +UpdateExam(ctx, exam, userID) error
        +ScheduleExam(ctx, id, enterpriseID, startTime, endTime, userID) error
        +CloneExam(ctx, sourceID, enterpriseID, title, userID) Exam
        +PublishExam(ctx, id, enterpriseID) error
        +CloseExam(ctx, id, enterpriseID) error
        +DeleteExam(ctx, id, enterpriseID) error
        +AddQuestionToExam(ctx, enterpriseID, examID, questionID) ExamQuestion
        +GetExamQuestions(ctx, examID, enterpriseID) List~ExamQuestion~
        +RemoveQuestionFromExam(ctx, enterpriseID, examID, questionID) error
        +AddRandomizationRule(ctx, enterpriseID, examID, topic, difficulty) ExamRandomizationRule
    }

    class QuestionUsecase {
        <<interface>>
        +CreateQuestion(ctx, q, userID) Question
        +GetQuestions(ctx, enterpriseID) List~Question~
        +GetQuestion(ctx, id, enterpriseID) Question
        +UpdateQuestion(ctx, q, userID) error
        +DeleteQuestion(ctx, id, enterpriseID) error
    }

    Question --> QuestionType : has
    Question --> DifficultyLevel : has
    Question "1" *-- "0..*" QuestionOption : contains
    Exam --> ExamStatus : has
    Exam "1" *-- "0..*" ExamQuestion : contains
    Exam "1" *-- "0..*" ExamRandomizationRule : contains
    ExamQuestion --> Question : references
    ExamRandomizationRule --> DifficultyLevel : filtered by
    QuestionRepository ..> Question : manages
    ExamRepository ..> Exam : manages
    ExamUsecase ..> ExamRepository : uses
    QuestionUsecase ..> QuestionRepository : uses
    ExamUsecase ..> QuestionRepository : uses
```

---

## Candidate Service

```mermaid
classDiagram
    direction TB

    class CandidateProfile {
        +UUID ID
        +UUID EnterpriseID
        +string ExternalID
        +string FirstName
        +string LastName
        +string Email
        +string FaceReferenceURL
        +bool IsActive
        +time CreatedAt
    }

    class ExamEnrollment {
        +UUID ID
        +UUID EnterpriseID
        +UUID ExamID
        +UUID CandidateID
        +string InvitationMethod
        +string AccessTokenHash
        +time TokenExpiresAt
        +int MaxAttempts
        +int AttemptsUsed
        +string Status
        +time CreatedAt
    }

    class ExamSession {
        +UUID ID
        +UUID EnterpriseID
        +UUID ExamID
        +UUID CandidateID
        +UUID EnrollmentID
        +SessionStatus Status
        +time StartedAt
        +time ExpiresAt
        +time SubmittedAt
        +time TerminatedAt
        +string TerminationReason
        +string ClientIP
        +string UserAgent
        +string FaceRegisteredURL
        +float64 CheatingScore
        +time CreatedAt
    }

    class SessionStatus {
        <<enumeration>>
        Active
        Submitted
        Terminated
        Expired
    }

    class SessionQuestion {
        +UUID ID
        +UUID SessionID
        +UUID QuestionID
        +JSON QuestionSnapshot
        +int OrderIndex
        +int Points
        +float64 NegativePoints
    }

    class SessionAnswer {
        +UUID ID
        +UUID SessionID
        +UUID SessionQuestionID
        +JSON AnswerData
        +bool IsFinal
        +time SavedAt
    }

    class ExamSubmission {
        +UUID ID
        +UUID SessionID
        +time SubmittedAt
        +bool AutoSubmitted
        +float64 TotalScore
        +string GradingStatus
        +time CreatedAt
    }

    class CandidateRepository {
        <<interface>>
        +Create(ctx, candidate) error
        +CreateBulk(ctx, candidates) error
        +GetByID(ctx, id, enterpriseID) CandidateProfile
        +GetByExternalID(ctx, externalID, enterpriseID) CandidateProfile
        +ListByEnterprise(ctx, enterpriseID) List~CandidateProfile~
        +Update(ctx, candidate) error
        +Deactivate(ctx, id, enterpriseID) error
    }

    class EnrollmentRepository {
        <<interface>>
        +Create(ctx, enrollment) error
        +GetByID(ctx, id, enterpriseID) ExamEnrollment
        +GetByExamAndCandidate(ctx, examID, candidateID) ExamEnrollment
        +ListByExam(ctx, examID, enterpriseID) List~ExamEnrollment~
        +Update(ctx, enrollment) error
        +IncrementAttempt(ctx, id) error
    }

    class SessionRepository {
        <<interface>>
        +CreateSession(ctx, session) error
        +GetSessionByID(ctx, id) ExamSession
        +ListSessionsByExam(ctx, examID, status) List~ExamSession~
        +UpdateSessionStatus(ctx, id, status, reason) error
        +SaveQuestionsSnapshot(ctx, sessionID, questions) error
        +GetSessionQuestions(ctx, sessionID) List~SessionQuestion~
        +UpsertAnswer(ctx, answer) error
        +GetSessionAnswers(ctx, sessionID) List~SessionAnswer~
        +CreateSubmission(ctx, submission) error
        +GetSubmissionBySession(ctx, sessionID) ExamSubmission
    }

    class CandidateUseCase {
        <<interface>>
        +CreateCandidate(ctx, candidate) CandidateProfile
        +BulkUpload(ctx, enterpriseID, csvData) int
        +GetCandidates(ctx, enterpriseID) List~CandidateProfile~
        +GetCandidate(ctx, id, enterpriseID) CandidateProfile
        +UpdateCandidate(ctx, candidate) error
        +DeactivateCandidate(ctx, id, enterpriseID) error
    }

    class EnrollmentUseCase {
        <<interface>>
        +EnrollCandidates(ctx, enterpriseID, examID, candidateIDs) List~string~
        +GetEnrollmentsForExam(ctx, examID, enterpriseID) List~ExamEnrollment~
        +GetEnrollment(ctx, id, enterpriseID) ExamEnrollment
        +RegenerateToken(ctx, id, enterpriseID) string
        +RevokeEnrollment(ctx, id, enterpriseID) error
        +ResetAttempts(ctx, id, enterpriseID) error
    }

    class SessionUseCase {
        <<interface>>
        +ValidateAccessToken(ctx, token) map
        +StartSession(ctx, token, clientIP, userAgent) ExamSession
        +ResumeActiveSession(ctx, candidateID) ExamSession
        +GetSessionDetails(ctx, sessionID, userID, role) ExamSession
        +GetSessionQuestionsSnapshot(ctx, sessionID, candidateID) List~SessionQuestion~
        +SaveAnswers(ctx, sessionID, candidateID, questionID, answerData) error
        +SubmitExam(ctx, sessionID, candidateID, autoSubmitted) ExamSubmission
        +TerminateSession(ctx, sessionID, enterpriseID, reason) error
    }

    class MonitoringUseCase {
        <<interface>>
        +ListSessionsForExam(ctx, examID, enterpriseID) List~ExamSession~
        +GetSessionSummary(ctx, sessionID, enterpriseID) ExamSession
        +GetSubmissions(ctx, examID, enterpriseID) List~ExamSubmission~
        +CandidateGetResult(ctx, sessionID, candidateID) ExamSubmission
    }

    ExamEnrollment --> CandidateProfile : references
    ExamSession --> ExamEnrollment : originates from
    ExamSession --> SessionStatus : has
    ExamSession "1" *-- "0..*" SessionQuestion : contains
    ExamSession "1" *-- "0..*" SessionAnswer : collects
    ExamSession "1" *-- "0..1" ExamSubmission : produces
    SessionAnswer --> SessionQuestion : responds to
    CandidateRepository ..> CandidateProfile : manages
    EnrollmentRepository ..> ExamEnrollment : manages
    SessionRepository ..> ExamSession : manages
    CandidateUseCase ..> CandidateRepository : uses
    EnrollmentUseCase ..> EnrollmentRepository : uses
    SessionUseCase ..> SessionRepository : uses
    MonitoringUseCase ..> SessionRepository : uses
```

---

## Payment Service

```mermaid
classDiagram
    direction TB

    class SubscriptionPlan {
        +UUID ID
        +string Name
        +string Slug
        +string Description
        +float64 Price
        +Currency Currency
        +BillingCycle BillingCycle
        +map Features
        +string StripePriceID
        +bool IsActive
        +time CreatedAt
        +time UpdatedAt
    }

    class EnterpriseSubscription {
        +UUID ID
        +UUID EnterpriseID
        +UUID PlanID
        +SubscriptionStatus Status
        +time CurrentPeriodStart
        +time CurrentPeriodEnd
        +bool CancelAtPeriodEnd
        +time CanceledAt
        +time EndedAt
        +string StripeCustomerID
        +string StripeSubscriptionID
        +time CreatedAt
        +time UpdatedAt
    }

    class Invoice {
        +UUID ID
        +UUID EnterpriseID
        +UUID SubscriptionID
        +string Number
        +InvoiceStatus Status
        +float64 AmountDue
        +float64 AmountPaid
        +float64 AmountRemaining
        +Currency Currency
        +time DueDate
        +time PaidAt
        +string HostedInvoiceURL
        +string InvoicePDFURL
        +time CreatedAt
        +time UpdatedAt
    }

    class Payment {
        +UUID ID
        +UUID EnterpriseID
        +UUID InvoiceID
        +float64 Amount
        +Currency Currency
        +PaymentStatus Status
        +string PaymentMethodType
        +string Provider
        +string ProviderPaymentID
        +string ProviderErrorCode
        +string ProviderErrorMessage
        +time CreatedAt
    }

    class BillingCycle {
        <<enumeration>>
        monthly
        yearly
    }

    class Currency {
        <<enumeration>>
        ETB
        USD
        EUR
        GBP
    }

    class SubscriptionStatus {
        <<enumeration>>
        Active
        PastDue
        Canceled
        Expired
        Trial
    }

    class InvoiceStatus {
        <<enumeration>>
        Draft
        Open
        Paid
        Void
        Uncollectible
    }

    class PaymentStatus {
        <<enumeration>>
        Pending
        Succeeded
        Failed
        Refunded
    }

    EnterpriseSubscription --> SubscriptionPlan : subscribes to
    EnterpriseSubscription --> SubscriptionStatus : has
    Invoice --> EnterpriseSubscription : billed under
    Invoice --> InvoiceStatus : has
    Payment --> Invoice : settles
    Payment --> PaymentStatus : has
    SubscriptionPlan --> BillingCycle : has
    SubscriptionPlan --> Currency : priced in
```

---

## Cross-Service Relationships

```mermaid
classDiagram
    direction LR

    class AuthUser["auth-service · User"] {
        +UUID ID
        +UUID EnterpriseID
        +Role Role
    }

    class EnterpriseService["enterprise-service · Enterprise"] {
        +UUID ID
        +UUID OwnerAccountID
        +SubscriptionStatus SubscriptionStatus
    }

    class ExamService["exam-service · Exam"] {
        +UUID ID
        +UUID EnterpriseID
        +UUID CreatedBy
    }

    class CandidateService["candidate-service · CandidateProfile"] {
        +UUID ID
        +UUID EnterpriseID
    }

    class PaymentService["payment-service · EnterpriseSubscription"] {
        +UUID ID
        +UUID EnterpriseID
        +UUID PlanID
    }

    class SessionService["candidate-service · ExamSession"] {
        +UUID ExamID
        +UUID CandidateID
        +UUID EnrollmentID
    }

    AuthUser ..> EnterpriseService : belongs to (EnterpriseID)
    EnterpriseService ..> PaymentService : has subscription (EnterpriseID)
    ExamService ..> EnterpriseService : scoped to (EnterpriseID)
    CandidateService ..> EnterpriseService : registered under (EnterpriseID)
    SessionService ..> ExamService : runs exam (ExamID)
    SessionService ..> CandidateService : for candidate (CandidateID)
```
