# Veritas — System Class Diagram

> Class diagram derived from both the Go domain models and the SQL migration files for each service.
> Table names (e.g. `veritas_users`) are shown in class titles.
> `<<interface>>` = Go port/usecase interface · `<<enumeration>>` = Go const block / SQL ENUM
> Cross-service FK references are annotated on arrows; they cross service boundaries and are not always enforced by DB FK in the consuming service.

---

## Auth Service

> `veritas_users` is **created** by the enterprise-service migration and **read** by the auth-service. The auth-service only owns `refresh_tokens`.

```mermaid
classDiagram
    direction TB

    class User["User · veritas_users"] {
        +UUID id PK
        +VARCHAR_255 email UNIQUE NOT NULL
        +TEXT password_hash NOT NULL
        +VARCHAR_50 honorific
        +VARCHAR_255 first_name
        +VARCHAR_255 last_name
        +VARCHAR_50 phone
        +VARCHAR_50 role NOT NULL
        +UUID enterprise_id FK NULL
        +BOOLEAN is_active DEFAULT_true
        +BOOLEAN is_deleted DEFAULT_false
        +BOOLEAN email_verified DEFAULT_false
        +TIMESTAMPTZ email_verified_at
        +INT failed_login_attempts DEFAULT_0
        +TIMESTAMPTZ locked_until
        +TIMESTAMPTZ password_changed_at
        +BOOLEAN must_change_password DEFAULT_false
        +TIMESTAMPTZ last_login_at
        +INET last_login_ip
        +TEXT last_user_agent
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
    }

    class RefreshToken["RefreshToken · refresh_tokens"] {
        +UUID id PK
        +UUID user_id FK NOT NULL
        +TEXT token_hash UNIQUE NOT NULL
        +TIMESTAMPTZ expires_at NOT NULL
        +BOOLEAN revoked DEFAULT_false
        +TIMESTAMPTZ created_at
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
        +GenerateRefreshToken() rawToken and tokenHash
    }

    User --> Role : role field
    RefreshToken --> User : user_id ON DELETE CASCADE
    UserRepository ..> User : queries
    RefreshTokenRepository ..> RefreshToken : manages
    TokenService ..> User : signs claims for
```

**Indexes:** `idx_refresh_tokens_user_id`, `idx_refresh_tokens_token_hash`
**FK:** `refresh_tokens.user_id` → `veritas_users(id)` ON DELETE CASCADE

---

## Enterprise Service

```mermaid
classDiagram
    direction TB

    class User["User · veritas_users"] {
        +UUID id PK
        +VARCHAR_255 email UNIQUE NOT NULL
        +TEXT password_hash NOT NULL
        +VARCHAR_50 honorific
        +VARCHAR_255 first_name
        +VARCHAR_255 last_name
        +VARCHAR_50 phone
        +VARCHAR_50 role NOT NULL
        +UUID enterprise_id FK NULL
        +BOOLEAN is_active DEFAULT_true
        +BOOLEAN is_deleted DEFAULT_false
        +BOOLEAN email_verified DEFAULT_false
        +TIMESTAMPTZ email_verified_at
        +INT failed_login_attempts DEFAULT_0
        +TIMESTAMPTZ locked_until
        +TIMESTAMPTZ password_changed_at
        +BOOLEAN must_change_password DEFAULT_false
        +TIMESTAMPTZ last_login_at
        +INET last_login_ip
        +TEXT last_user_agent
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
    }

    class Enterprise["Enterprise · veritas_enterprise"] {
        +UUID id PK
        +VARCHAR_120 slug UNIQUE NOT NULL
        +VARCHAR_255 display_name NOT NULL
        +VARCHAR_255 legal_name NOT NULL
        +VARCHAR_255 contact_email NOT NULL
        +UUID owner_account_id FK NOT NULL
        +enterprise_status status DEFAULT_PendingApproval
        +TIMESTAMPTZ approved_at
        +TIMESTAMPTZ suspended_at
        +TIMESTAMPTZ deleted_at
        +TIMESTAMPTZ retention_until
        +UUID subscription_plan_id NULL
        +subscription_status subscription_status NULL
        +TIMESTAMPTZ current_period_start
        +TIMESTAMPTZ current_period_end
        +TEXT logo_url
        +VARCHAR_7 primary_color
        +VARCHAR_7 secondary_color
        +VARCHAR_255 custom_domain UNIQUE
        +VARCHAR_50 contact_phone
        +VARCHAR_255 address_line1
        +VARCHAR_255 address_line2
        +VARCHAR_100 city
        +VARCHAR_100 country
        +JSONB settings DEFAULT_empty
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
        +UUID created_by NOT NULL
        +UUID updated_by NOT NULL
    }

    class AuditLog["AuditLog · veritas_enterprise_audit_logs"] {
        +UUID id PK
        +UUID enterprise_id FK NOT NULL
        +UUID actor_id NOT NULL
        +VARCHAR_50 actor_role NOT NULL
        +VARCHAR_100 event NOT NULL
        +JSONB metadata DEFAULT_empty
        +TIMESTAMPTZ created_at
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

    Enterprise --> EnterpriseStatus : status ENUM
    Enterprise --> SubscriptionStatus : subscription_status ENUM
    Enterprise --> User : owner_account_id ON DELETE RESTRICT
    AuditLog --> Enterprise : enterprise_id ON DELETE CASCADE
    EnterpriseRepository ..> Enterprise : manages
    UserRepository ..> User : manages
    AuditRepository ..> AuditLog : manages
    EnterpriseUsecase ..> EnterpriseRepository : uses
    EnterpriseUsecase ..> AuditRepository : uses
    UserUsecase ..> UserRepository : uses
```

**Indexes:** `idx_enterprises_status`, `idx_enterprises_owner`, `idx_enterprises_subscription_status`,
`idx_enterprises_active` (partial WHERE Active), `idx_enterprises_subscription_expiry` (partial), `idx_enterprises_settings` (GIN),
`idx_veritas_users_email`, `idx_veritas_users_role`, `idx_veritas_users_enterprise_id` (partial)
**Constraint:** `chk_slug_format` — slug must match `^[a-z0-9-]+$`

---

## Exam Service

> Cross-service FKs to `veritas_enterprise` and `veritas_users` are enforced at the DB level in this service's migration.

```mermaid
classDiagram
    direction TB

    class Question["Question · veritas_questions"] {
        +UUID id PK
        +UUID enterprise_id FK NOT NULL
        +question_type type NOT NULL
        +VARCHAR_255 topic NOT NULL
        +difficulty_level difficulty NOT NULL
        +VARCHAR_500 title NOT NULL
        +TEXT content NOT NULL
        +TEXT media_url
        +INT points DEFAULT_1
        +NUMERIC_5_2 negative_points DEFAULT_0
        +JSONB metadata DEFAULT_empty
        +BOOLEAN is_active DEFAULT_true
        +UUID created_by FK NOT NULL
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
    }

    class QuestionOption["QuestionOption · veritas_question_options"] {
        +UUID id PK
        +UUID question_id FK NOT NULL
        +TEXT content NOT NULL
        +BOOLEAN is_correct DEFAULT_false
    }

    class Exam["Exam · veritas_exams"] {
        +UUID id PK
        +UUID enterprise_id FK NOT NULL
        +VARCHAR_255 title NOT NULL
        +TEXT description
        +INT duration_minutes NOT NULL
        +NUMERIC_5_2 passing_score_percent NOT NULL
        +BOOLEAN negative_marking DEFAULT_false
        +INT max_participants
        +VARCHAR_50 invitation_method NOT NULL
        +exam_status status DEFAULT_Draft
        +UUID template_source_id FK NULL
        +TIMESTAMPTZ scheduled_start
        +TIMESTAMPTZ scheduled_end
        +JSONB settings DEFAULT_empty
        +UUID created_by FK NOT NULL
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
    }

    class ExamQuestion["ExamQuestion · veritas_exam_questions"] {
        +UUID id PK
        +UUID exam_id FK NOT NULL
        +UUID question_id FK NOT NULL
        +INT points_override NULL
        +INT order_index NULL
    }

    class ExamRandomizationRule["ExamRandomizationRule · veritas_exam_randomization_rules"] {
        +UUID id PK
        +UUID exam_id FK NOT NULL
        +VARCHAR_255 topic NULL
        +difficulty_level difficulty NULL
        +INT question_count NOT NULL
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

    class ExamStatus {
        <<enumeration>>
        Draft
        Scheduled
        Active
        Closed
        Archived
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
        +ScheduleExam(ctx, id, startTime, endTime, userID) error
        +CloneExam(ctx, sourceID, enterpriseID, title, userID) Exam
        +PublishExam(ctx, id, enterpriseID) error
        +CloseExam(ctx, id, enterpriseID) error
        +AddQuestionToExam(ctx, ...) ExamQuestion
        +GetExamQuestions(ctx, ...) List~ExamQuestion~
        +AddRandomizationRule(ctx, ...) ExamRandomizationRule
    }

    class QuestionUsecase {
        <<interface>>
        +CreateQuestion(ctx, q, userID) Question
        +GetQuestions(ctx, enterpriseID) List~Question~
        +GetQuestion(ctx, id, enterpriseID) Question
        +UpdateQuestion(ctx, q, userID) error
        +DeleteQuestion(ctx, id, enterpriseID) error
    }

    Question --> QuestionType : type ENUM
    Question --> DifficultyLevel : difficulty ENUM
    Question "1" *-- "0..*" QuestionOption : ON DELETE CASCADE
    Exam --> ExamStatus : status ENUM
    Exam --> Exam : template_source_id self-ref ON DELETE SET NULL
    Exam "1" *-- "0..*" ExamQuestion : ON DELETE CASCADE
    Exam "1" *-- "0..*" ExamRandomizationRule : ON DELETE CASCADE
    ExamQuestion --> Question : question_id ON DELETE CASCADE
    ExamRandomizationRule --> DifficultyLevel : difficulty ENUM
    QuestionRepository ..> Question : manages
    ExamRepository ..> Exam : manages
    ExamUsecase ..> ExamRepository : uses
    QuestionUsecase ..> QuestionRepository : uses
    ExamUsecase ..> QuestionRepository : uses
```

**Unique constraint:** `(exam_id, question_id)` on `veritas_exam_questions`
**Indexes:** `idx_questions_enterprise`, `idx_exams_enterprise`, `idx_exams_status`, `idx_exam_schedule`,
`idx_question_options_question`, `idx_exam_questions_exam`, `idx_random_rules_exam`

---

## Candidate Service

```mermaid
classDiagram
    direction TB

    class CandidateProfile["CandidateProfile · candidate_profiles"] {
        +UUID id PK
        +UUID enterprise_id NOT NULL
        +VARCHAR_255 external_id NOT NULL
        +VARCHAR_255 first_name NOT NULL
        +VARCHAR_255 last_name NOT NULL
        +VARCHAR_255 email NULL
        +TEXT face_reference_url NULL
        +BOOLEAN is_active DEFAULT_true
        +TIMESTAMPTZ created_at
    }

    class ExamEnrollment["ExamEnrollment · exam_enrollments"] {
        +UUID id PK
        +UUID enterprise_id NOT NULL
        +UUID exam_id NOT NULL
        +UUID candidate_id FK NOT NULL
        +VARCHAR_50 invitation_method NOT NULL
        +TEXT access_token_hash NOT NULL
        +TIMESTAMPTZ token_expires_at NOT NULL
        +INT max_attempts DEFAULT_1
        +INT attempts_used DEFAULT_0
        +VARCHAR_50 status DEFAULT_Invited
        +TIMESTAMPTZ created_at
    }

    class ExamSession["ExamSession · exam_sessions"] {
        +UUID id PK
        +UUID enterprise_id NOT NULL
        +UUID exam_id NOT NULL
        +UUID candidate_id FK NOT NULL
        +UUID enrollment_id FK NOT NULL
        +session_status status DEFAULT_Active
        +TIMESTAMPTZ started_at DEFAULT_NOW
        +TIMESTAMPTZ expires_at NOT NULL
        +TIMESTAMPTZ submitted_at NULL
        +TIMESTAMPTZ terminated_at NULL
        +TEXT termination_reason NULL
        +INET client_ip NULL
        +TEXT user_agent NULL
        +TEXT face_registered_url NULL
        +NUMERIC_5_2 cheating_score NULL
        +TIMESTAMPTZ created_at
    }

    class SessionQuestion["SessionQuestion · session_questions"] {
        +UUID id PK
        +UUID session_id FK NOT NULL
        +UUID question_id NOT NULL
        +JSONB question_snapshot NOT NULL
        +INT order_index NOT NULL
        +INT points NOT NULL
        +NUMERIC_5_2 negative_points DEFAULT_0
    }

    class SessionAnswer["SessionAnswer · session_answers"] {
        +UUID id PK
        +UUID session_id FK NOT NULL
        +UUID session_question_id FK NOT NULL
        +JSONB answer_data NOT NULL
        +BOOLEAN is_final DEFAULT_false
        +TIMESTAMPTZ saved_at DEFAULT_NOW
    }

    class ExamSubmission["ExamSubmission · exam_submissions"] {
        +UUID id PK
        +UUID session_id FK UNIQUE NOT NULL
        +TIMESTAMPTZ submitted_at NOT NULL
        +BOOLEAN auto_submitted DEFAULT_false
        +NUMERIC_6_2 total_score NULL
        +VARCHAR_50 grading_status DEFAULT_Pending
        +TIMESTAMPTZ created_at
    }

    class SessionStatus {
        <<enumeration>>
        Active
        Submitted
        Terminated
        Expired
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
        +SaveAnswers(ctx, sessionID, candidateID, questionID, data) error
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

    ExamEnrollment --> CandidateProfile : candidate_id ON DELETE CASCADE
    ExamSession --> CandidateProfile : candidate_id ON DELETE CASCADE
    ExamSession --> ExamEnrollment : enrollment_id ON DELETE CASCADE
    ExamSession --> SessionStatus : status ENUM
    ExamSession "1" *-- "0..*" SessionQuestion : session_id ON DELETE CASCADE
    ExamSession "1" *-- "0..*" SessionAnswer : session_id ON DELETE CASCADE
    ExamSession "1" *-- "0..1" ExamSubmission : session_id UNIQUE ON DELETE CASCADE
    SessionAnswer --> SessionQuestion : session_question_id ON DELETE CASCADE
    CandidateRepository ..> CandidateProfile : manages
    EnrollmentRepository ..> ExamEnrollment : manages
    SessionRepository ..> ExamSession : manages
    CandidateUseCase ..> CandidateRepository : uses
    EnrollmentUseCase ..> EnrollmentRepository : uses
    SessionUseCase ..> SessionRepository : uses
    MonitoringUseCase ..> SessionRepository : uses
```

**Unique constraints:** `(enterprise_id, external_id)` on `candidate_profiles`; `(exam_id, candidate_id)` on `exam_enrollments`;
`session_id` on `exam_submissions`; `(session_id, session_question_id)` on `session_answers`
**Indexes:** `idx_candidate_enterprise`, `idx_enrollment_exam`, `idx_enrollment_candidate`, `idx_enrollment_status`,
`idx_session_exam`, `idx_session_candidate`, `idx_session_enrollment`, `idx_session_status`,
`idx_sq_session`, `idx_answers_session`, `idx_submissions_session`

---

## Payment Service

```mermaid
classDiagram
    direction TB

    class SubscriptionPlan["SubscriptionPlan · veritas_subscription_plans"] {
        +UUID id PK
        +VARCHAR_100 name UNIQUE NOT NULL
        +VARCHAR_100 slug UNIQUE NOT NULL
        +TEXT description
        +DECIMAL_12_2 price DEFAULT_0
        +currency_type currency DEFAULT_ETB
        +billing_cycle_type billing_cycle DEFAULT_monthly
        +JSONB features DEFAULT_empty
        +BOOLEAN is_active DEFAULT_true
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
    }

    class EnterpriseSubscription["EnterpriseSubscription · veritas_enterprise_subscriptions"] {
        +UUID id PK
        +UUID enterprise_id UNIQUE NOT NULL
        +UUID plan_id FK NOT NULL
        +sub_status_type status DEFAULT_Active
        +TIMESTAMPTZ current_period_start DEFAULT_NOW
        +TIMESTAMPTZ current_period_end NOT NULL
        +BOOLEAN cancel_at_period_end DEFAULT_false
        +TIMESTAMPTZ canceled_at NULL
        +TIMESTAMPTZ ended_at NULL
        +VARCHAR_255 stripe_customer_id NULL
        +VARCHAR_255 stripe_subscription_id NULL
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
    }

    class Invoice["Invoice · veritas_invoices"] {
        +UUID id PK
        +UUID enterprise_id FK NOT NULL
        +UUID subscription_id FK NOT NULL
        +VARCHAR_50 number UNIQUE NOT NULL
        +invoice_status_type status DEFAULT_Draft
        +DECIMAL_12_2 amount_due NOT NULL
        +DECIMAL_12_2 amount_paid DEFAULT_0
        +DECIMAL_12_2 amount_remaining NOT NULL
        +currency_type currency DEFAULT_ETB
        +TIMESTAMPTZ due_date NOT NULL
        +TIMESTAMPTZ paid_at NULL
        +TEXT hosted_invoice_url NULL
        +TEXT invoice_pdf_url NULL
        +TIMESTAMPTZ created_at
        +TIMESTAMPTZ updated_at
    }

    class Payment["Payment · veritas_payments"] {
        +UUID id PK
        +UUID enterprise_id FK NOT NULL
        +UUID invoice_id FK NULL
        +DECIMAL_12_2 amount NOT NULL
        +currency_type currency DEFAULT_ETB
        +payment_status_type status NOT NULL
        +VARCHAR_50 payment_method_type NULL
        +VARCHAR_50 provider NOT NULL
        +VARCHAR_255 provider_payment_id UNIQUE NOT NULL
        +VARCHAR_100 provider_error_code NULL
        +TEXT provider_error_message NULL
        +TIMESTAMPTZ created_at
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

    EnterpriseSubscription --> SubscriptionPlan : plan_id ON DELETE RESTRICT
    EnterpriseSubscription --> SubscriptionStatus : status ENUM
    Invoice --> EnterpriseSubscription : enterprise_id and subscription_id ON DELETE CASCADE
    Invoice --> InvoiceStatus : status ENUM
    Payment --> EnterpriseSubscription : enterprise_id ON DELETE CASCADE
    Payment --> Invoice : invoice_id ON DELETE SET NULL
    Payment --> PaymentStatus : status ENUM
    SubscriptionPlan --> BillingCycle : billing_cycle ENUM
    SubscriptionPlan --> Currency : currency ENUM
```

**Unique constraints:** `enterprise_id` on `veritas_enterprise_subscriptions` (one per enterprise);
`provider_payment_id` on `veritas_payments`; `number` on `veritas_invoices`
**FK note:** `veritas_invoices.enterprise_id` references `veritas_enterprise_subscriptions(enterprise_id)`, not `veritas_enterprise(id)`
**Indexes:** `idx_subs_enterprise`, `idx_subs_status`, `idx_invoices_enterprise`, `idx_invoices_status`,
`idx_payments_enterprise`, `idx_payments_status`, `idx_plans_slug`

---

## Cross-Service Relationships

```mermaid
classDiagram
    direction LR

    class AuthService["auth-service"] {
        reads veritas_users
        owns refresh_tokens
    }

    class EnterpriseService["enterprise-service"] {
        owns veritas_users
        owns veritas_enterprise
        owns veritas_enterprise_audit_logs
    }

    class ExamService["exam-service"] {
        owns veritas_questions
        owns veritas_question_options
        owns veritas_exams
        owns veritas_exam_questions
        owns veritas_exam_randomization_rules
        DB-FK to veritas_enterprise and veritas_users
    }

    class CandidateService["candidate-service"] {
        owns candidate_profiles
        owns exam_enrollments
        owns exam_sessions
        owns session_questions
        owns session_answers
        owns exam_submissions
        soft-ref exam_id to exam-service
    }

    class PaymentService["payment-service"] {
        owns veritas_subscription_plans
        owns veritas_enterprise_subscriptions
        owns veritas_invoices
        owns veritas_payments
        soft-ref enterprise_id to enterprise-service
    }

    AuthService ..> EnterpriseService : reads veritas_users
    ExamService ..> EnterpriseService : FK enterprise_id and created_by
    CandidateService ..> EnterpriseService : enterprise_id no DB FK
    CandidateService ..> ExamService : exam_id no DB FK
    PaymentService ..> EnterpriseService : enterprise_id no DB FK
    EnterpriseService ..> PaymentService : subscription_plan_id ref
```
