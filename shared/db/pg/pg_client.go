package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresClient interface {
	QueryRow(ctx context.Context, query string, args ...any) Row
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	Exec(ctx context.Context, query string, args ...any) (int, error)
	BeginTx(ctx context.Context) (Tx, error)
	LogConnectionInfo()
	Close()
}

type Tx interface {
	QueryRow(ctx context.Context, query string, args ...any) Row
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	Exec(ctx context.Context, query string, args ...any) (int, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

type pgClient struct {
	pool *pgxpool.Pool
}

// BeginTx implements PostgresClient.
func (p *pgClient) BeginTx(ctx context.Context) (Tx, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	return &pgTx{tx: tx}, nil
}

func (p *pgClient) LogConnectionInfo() {
	config := p.pool.Config()
	fmt.Printf("Connected to DB Host: %s, Port: %d, Database: %s, User: %s\n",
		config.ConnConfig.Host,
		config.ConnConfig.Port,
		config.ConnConfig.Database,
		config.ConnConfig.User,
	)
}

// --- Row ---
type pgRow struct {
	row pgx.Row
}

// --- Rows ---
type pgRows struct {
	rows pgx.Rows
}

// --- Tx ---
type pgTx struct {
	tx pgx.Tx
}

// Commit implements Tx.
func (p *pgTx) Commit(ctx context.Context) error {
	return p.tx.Commit(ctx)
}

// Exec implements Tx.
func (p *pgTx) Exec(ctx context.Context, query string, args ...any) (int, error) {
	commandTag, err := p.tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return int(commandTag.RowsAffected()), nil
}

// Query implements Tx.
func (p *pgTx) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := p.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgRows{rows: rows}, nil
}

// QueryRow implements Tx.
func (p *pgTx) QueryRow(ctx context.Context, query string, args ...any) Row {
	return &pgRow{row: p.tx.QueryRow(ctx, query, args...)}
}

// Rollback implements Tx.
func (p *pgTx) Rollback(ctx context.Context) error {
	return p.tx.Rollback(ctx)
}

func NewPostgresClient(dsn string) (PostgresClient, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return &pgClient{pool: pool}, nil
}

func (p *pgClient) QueryRow(ctx context.Context, query string, args ...any) Row {
	return &pgRow{row: p.pool.QueryRow(ctx, query, args...)}
}

func (p *pgClient) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgRows{rows: rows}, nil
}

func (p *pgClient) Exec(ctx context.Context, query string, args ...any) (int, error) {
	commandTag, err := p.pool.Exec(ctx, query, args...)
	return int(commandTag.RowsAffected()), err
}

func (p *pgClient) Close() {
	p.pool.Close()
}

// --- Row ---
func (r *pgRow) Scan(dest ...any) error {
	return r.row.Scan(dest...)
}

// --- Rows ---
func (r *pgRows) Next() bool {
	return r.rows.Next()
}

func (r *pgRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *pgRows) Close() error {
	r.rows.Close()
	return nil
}

func (r *pgRows) Err() error {
	return r.rows.Err()
}
