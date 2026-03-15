package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

type auditRepository struct {
	db DBTX
}

// NewAuditRepository creates a new audit repository.
func NewAuditRepository(db DBTX) domain.AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) WithTx(tx pgx.Tx) domain.AuditRepository {
	return &auditRepository{db: tx}
}

// Create inserts a new audit log record.
func (r *auditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	const query = `
		INSERT INTO veritas_enterprise_audit_logs
		  (id, enterprise_id, actor_id, actor_role, event, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	metadataJson, err := json.Marshal(log.Metadata)
	if err != nil {
		return err
	}

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	_, err = r.db.Exec(ctx, query,
		log.ID, log.EnterpriseID, log.ActorID, log.ActorRole, log.Event, string(metadataJson), log.CreatedAt,
	)
	return err
}

// ListByEnterprise returns paginated audit logs for an enterprise, newest first.
func (r *auditRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, page, limit int) ([]*domain.AuditLog, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM veritas_enterprise_audit_logs WHERE enterprise_id = $1",
		enterpriseID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	const fields = "id, enterprise_id, actor_id, actor_role, event, metadata, created_at"
	dataQ := fmt.Sprintf(
		"SELECT %s FROM veritas_enterprise_audit_logs WHERE enterprise_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		fields,
	)
	rows, err := r.db.Query(ctx, dataQ, enterpriseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(&l.ID, &l.EnterpriseID, &l.ActorID, &l.ActorRole, &l.Event, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, total, rows.Err()
}
