package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/dto"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type EnrollmentHandler struct {
	uc domain.EnrollmentUseCase
}

func NewEnrollmentHandler(uc domain.EnrollmentUseCase) *EnrollmentHandler {
	return &EnrollmentHandler{uc: uc}
}

// Enroll enrolls one or more candidates into an exam.
// This creates enrollment records and generates invitation URLs but does NOT
// send any emails. The admin must call NotifyAll or Notify to dispatch emails.
//
//	@Summary		Enroll candidates
//	@Description	Create enrollments for an exam. Returns per-candidate invitation URLs (opaque code, not JWT).
//	@Tags			enrollment
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string						false	"Enterprise ID"
//	@Param			examId			path		string						true	"Exam ID (UUID)"
//	@Param			body			body		dto.EnrollmentRequest		true	"Enrollment payload"
//	@Success		201				{object}	dto.EnrollmentCreateResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/enrollments [post]
func (h *EnrollmentHandler) Enroll(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	examID, err := uuid.Parse(c.Param("examId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	var req dto.EnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	results, err := h.uc.EnrollCandidates(c.Request.Context(), entID, examID, req.CandidateIDs, req.MaxAttempts, req.TokenExpiresAt)
	if err != nil {
		HandleError(c, err)
		return
	}

	items := make([]dto.EnrollmentResultItem, 0, len(results))
	for _, r := range results {
		items = append(items, dto.EnrollmentResultItem{
			EnrollmentID:  r.EnrollmentID,
			CandidateID:   r.CandidateID,
			InvitationURL: r.InvitationURL,
			Status:        string(r.Status),
		})
	}

	c.JSON(http.StatusCreated, dto.EnrollmentCreateResponse{
		Message: "Candidates enrolled successfully. Use /notify to send invitation emails.",
		Results: items,
	})
}

// NotifyAll sends invitation emails to all candidates enrolled in an exam.
// An optional list of enrollment IDs can be provided; if omitted, all
// Pending/Invited enrollments for the exam are notified.
//
//	@Summary		Bulk notify candidates
//	@Description	Send invitation emails to enrolled candidates. Admin-triggered; no emails are sent automatically on enroll.
//	@Tags			enrollment
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string					false	"Enterprise ID"
//	@Param			examId			path		string					true	"Exam ID (UUID)"
//	@Param			body			body		dto.NotifyBulkRequest	false	"Optional list of enrollment IDs to notify"
//	@Success		200				{object}	dto.NotifyResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/enrollments/notify [post]
func (h *EnrollmentHandler) NotifyAll(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	examID, err := uuid.Parse(c.Param("examId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	var req dto.NotifyBulkRequest
	_ = c.ShouldBindJSON(&req)
	enrollmentIDs := req.EnrollmentIDs
	if len(enrollmentIDs) == 0 {
		allEnrollments, _, err := h.uc.GetEnrollmentsForExam(c.Request.Context(), examID, entID, pagination.Params{})
		if err != nil {
			HandleError(c, err)
			return
		}
		for _, e := range allEnrollments {
			if e.Status == domain.StatusPending || e.Status == domain.StatusInvited {
				enrollmentIDs = append(enrollmentIDs, e.ID)
			}
		}
	}

	results, err := h.uc.NotifyCandidates(c.Request.Context(), examID, entID, enrollmentIDs)
	if err != nil {
		HandleError(c, err)
		return
	}

	items := make([]dto.NotifyResultItem, 0, len(results))
	for _, r := range results {
		items = append(items, dto.NotifyResultItem{
			EnrollmentID: r.EnrollmentID,
			CandidateID:  r.CandidateID,
			NotifyStatus: r.NotifyStatus,
		})
	}

	c.JSON(http.StatusOK, dto.NotifyResponse{
		Message: "Notification dispatch complete",
		Results: items,
	})
}

// Notify sends an invitation email to a single enrolled candidate.
//
//	@Summary		Notify single candidate
//	@Description	Send or resend an invitation email to one candidate enrollment.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string	false	"Enterprise ID"
//	@Param			enrollmentId	path		string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.NotifyResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId}/notify [post]
func (h *EnrollmentHandler) Notify(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	result, err := h.uc.NotifyCandidate(c.Request.Context(), id, entID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NotifyResponse{
		Message: "Notification dispatch complete",
		Results: []dto.NotifyResultItem{{
			EnrollmentID: result.EnrollmentID,
			CandidateID:  result.CandidateID,
			NotifyStatus: result.NotifyStatus,
		}},
	})
}

// GetLink returns a fresh invitation URL for admin to distribute manually.
// This is intended for candidates without an email address.
// Calling this endpoint rotates the opaque code, invalidating any previous URL.
//
//	@Summary		Get invitation link
//	@Description	Generate a fresh invitation URL (opaque code) for manual distribution. Old URL is invalidated.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string	false	"Enterprise ID"
//	@Param			enrollmentId	path		string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.InvitationLinkResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId}/link [get]
func (h *EnrollmentHandler) GetLink(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	link, err := h.uc.GetInvitationLink(c.Request.Context(), id, entID)
	if err != nil {
		HandleError(c, err)
		return
	}

	enr, err := h.uc.GetEnrollment(c.Request.Context(), id, entID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.InvitationLinkResponse{
		EnrollmentID:   enr.ID,
		InvitationURL:  link,
		Status:         string(enr.Status),
		TokenExpiresAt: enr.TokenExpiresAt,
	})
}

// ListByExam lists enrollments for an exam with pagination.
//
//	@Summary		List enrollments by exam
//	@Description	List exam enrollments for the caller enterprise with pagination.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Param			page			query	int		false	"Page number (default 1)"
//	@Param			limit			query	int		false	"Page size (default 10, max 1000)"
//	@Param			sort			query	string	false	"Sort field: created_at|attempts_used"
//	@Param			sort_dir		query	string	false	"Sort direction: asc|desc (default desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.ExamEnrollment]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/enrollments [get]
func (h *EnrollmentHandler) ListByExam(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	examID, err := uuid.Parse(c.Param("examId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	params := pagination.ParseGin(c)
	list, total, err := h.uc.GetEnrollmentsForExam(c.Request.Context(), examID, entID, params)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pagination.NewPaginatedResponse(list, total, params))
}

// Get fetches enrollment details by enrollment ID.
//
//	@Summary		Get enrollment
//	@Description	Get one enrollment by ID.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID"
//	@Param			enrollmentId	path	string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.EnrollmentResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId} [get]
func (h *EnrollmentHandler) Get(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	enr, err := h.uc.GetEnrollment(c.Request.Context(), id, entID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.EnrollmentResponse{Data: enr})
}

// Revoke revokes an enrollment and invalidates the invitation link.
//
//	@Summary		Revoke enrollment
//	@Description	Revoke an enrollment. Any outstanding invitation link is immediately invalidated.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID"
//	@Param			enrollmentId	path	string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId}/revoke [patch]
func (h *EnrollmentHandler) Revoke(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	if err := h.uc.RevokeEnrollment(c.Request.Context(), id, entID); err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Enrollment revoked"})
}

// Delete deletes an enrollment if its status is Pending.
//
//	@Summary		Delete enrollment
//	@Description	Delete an enrollment. Only allowed if status is Pending.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID"
//	@Param			enrollmentId	path	string	true	"Enrollment ID (UUID)"
//	@Success		204				"No Content"
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId} [delete]
func (h *EnrollmentHandler) Delete(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	if err := h.uc.DeleteEnrollment(c.Request.Context(), id, entID); err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ResetAttempts resets the attempt counter for an enrollment.
//
//	@Summary		Reset enrollment attempts
//	@Description	Reset attempts_used to zero for an enrollment.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID"
//	@Param			enrollmentId	path	string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId}/reset-attempts [post]
func (h *EnrollmentHandler) ResetAttempts(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	if err := h.uc.ResetAttempts(c.Request.Context(), id, entID); err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Enrollment attempts reset"})
}

// RedeemCode exchanges a candidate's opaque invitation code for a raw JWT.
// The JWT is returned in the response body only — it is NEVER placed in a URL.
// The candidate frontend should store the JWT in memory or an httpOnly cookie
// and use it to call POST /sessions/start.
//
//	@Summary		Redeem invitation code
//	@Description	Exchange the opaque invitation code (from the URL query param) for a JWT. JWT is in the response body only.
//	@Tags			enrollment
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.RedeemRequest	true	"Opaque code from invitation URL"
//	@Success		200		{object}	dto.RedeemResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		429		{object}	dto.ErrorResponse
//	@Router			/access/redeem [post]
func (h *EnrollmentHandler) RedeemCode(c *gin.Context) {
	var req dto.RedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	rawToken, err := h.uc.RedeemInvitationCode(c.Request.Context(), req.Code)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.RedeemResponse{Token: rawToken})
}

