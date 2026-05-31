import json
import logging
import json
from uuid import UUID
from typing import Any, Optional, Dict, List, Tuple
import asyncpg

from app.grading.grader import ExamGradeReport, QuestionResult
from app.grading.security import calculate_row_checksum
from app.domain.models import GradingStatus

logger = logging.getLogger("grading.repository")


class DataTamperingError(Exception):
    """Raised when a grading result's cryptographic checksum mismatch is detected."""
    def __init__(self, session_id: UUID):
        self.session_id = session_id
        super().__init__(f"Database integrity violation: grading result for session {session_id} has been tampered with!")


class GradingRepository:
    def __init__(self, pool: asyncpg.Pool):
        self._pool = pool

    async def _set_audit_context(
        self,
        conn: asyncpg.Connection,
        actor_id: Optional[UUID],
        actor_role: str,
        ip_address: Optional[str],
        reason: Optional[str],
    ) -> None:
        """Set session-level variables for the audit trigger."""
        await conn.execute(
            """
            SELECT
                set_config('veritas.current_actor_id', $1, true),
                set_config('veritas.current_actor_role', $2, true),
                set_config('veritas.current_ip', $3, true),
                set_config('veritas.current_reason', $4, true)
            """,
            str(actor_id) if actor_id else "",
            actor_role or "system",
            ip_address or "",
            reason or "",
        )

    async def save_grading_report(
        self,
        report: ExamGradeReport,
        graded_by: str = "system",
        actor_id: Optional[UUID] = None,
        actor_role: str = "system",
        ip_address: Optional[str] = None,
        reason: Optional[str] = None,
    ) -> UUID:
        """
        Save or update an exam grading report within a secure transaction.
        Checks for existing records, validates current integrity (detects tampering),
        and updates scoring parameters with optimistic concurrency control.
        """
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                # Check if a grade result already exists for this session
                existing = await conn.fetchrow(
                    "SELECT id, total_max_points, total_awarded_points, percentage, row_checksum, version FROM grading_results WHERE session_id = $1",
                    UUID(report.session_id)
                )

                version = 1
                if existing:
                    # 1. Integrity check (Verify if database row was tampered)
                    stored_checksum = existing["row_checksum"]
                    expected_checksum = calculate_row_checksum(
                        session_id=report.session_id,
                        candidate_id=report.candidate_id,
                        total_max_points=float(existing["total_max_points"]),
                        total_awarded_points=float(existing["total_awarded_points"]),
                        percentage=float(existing["percentage"]),
                        version=existing["version"]
                    )
                    if stored_checksum != expected_checksum:
                        logger.error(
                            "Tampering detected! Session: %s. DB Checksum: %s, Expected: %s",
                            report.session_id, stored_checksum, expected_checksum
                        )
                        raise DataTamperingError(UUID(report.session_id))

                    # 2. Increment version for optimistic locking
                    version = existing["version"] + 1
                    grading_result_id = existing["id"]

                    # 3. Calculate new HMAC checksum
                    new_checksum = calculate_row_checksum(
                        session_id=report.session_id,
                        candidate_id=report.candidate_id,
                        total_max_points=report.total_max_points,
                        total_awarded_points=report.total_awarded_points,
                        percentage=report.percentage,
                        version=version
                    )

                    # 4. Set audit context and update
                    await self._set_audit_context(conn, actor_id, actor_role, ip_address, reason or "Re-grading exam")
                    result = await conn.execute(
                        """
                        UPDATE grading_results SET
                            total_max_points = $1,
                            total_awarded_points = $2,
                            percentage = $3,
                            status = $4,
                            graded_by = $5,
                            row_checksum = $6,
                            updated_at = NOW()
                        WHERE session_id = $7 AND version = $8
                        """,
                        report.total_max_points,
                        report.total_awarded_points,
                        report.percentage,
                        GradingStatus.graded.value,
                        graded_by,
                        new_checksum,
                        UUID(report.session_id),
                        existing["version"]
                    )
                    # If version conflict occurred:
                    if result == "UPDATE 0":
                        raise RuntimeError("Optimistic locking conflict during grade update. Please retry.")

                    # Delete old question results
                    await conn.execute(
                        "DELETE FROM grading_question_results WHERE grading_result_id = $1",
                        grading_result_id
                    )
                else:
                    # 1. Compute HMAC checksum for first version
                    new_checksum = calculate_row_checksum(
                        session_id=report.session_id,
                        candidate_id=report.candidate_id,
                        total_max_points=report.total_max_points,
                        total_awarded_points=report.total_awarded_points,
                        percentage=report.percentage,
                        version=version
                    )

                    # 2. Set audit context and insert
                    await self._set_audit_context(conn, actor_id, actor_role, ip_address, reason or "Initial automated grading")
                    grading_result_id = await conn.fetchval(
                        """
                        INSERT INTO grading_results (
                            session_id, exam_id, candidate_id, enterprise_id, enrollment_id,
                            total_max_points, total_awarded_points, percentage, status, graded_by,
                            row_checksum, version
                        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
                        RETURNING id
                        """,
                        UUID(report.session_id),
                        UUID(report.exam_id),
                        UUID(report.candidate_id),
                        UUID(report.enterprise_id),
                        UUID(report.enrollment_id),
                        report.total_max_points,
                        report.total_awarded_points,
                        report.percentage,
                        GradingStatus.graded.value,
                        graded_by,
                        new_checksum,
                        version
                    )

                # Insert question results
                for qr in report.question_results:
                    await conn.execute(
                        """
                        INSERT INTO grading_question_results (
                            grading_result_id, question_id, session_question_id,
                            question_type, title, content, candidate_answer, max_points, awarded_points, status
                        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                        """,
                        grading_result_id,
                        UUID(qr.question_id),
                        UUID(qr.session_question_id),
                        qr.question_type,
                        qr.title,
                        qr.content,
                        json.dumps(qr.candidate_answer) if qr.candidate_answer else None,
                        qr.max_points,
                        qr.awarded_points,
                        qr.status
                    )

                return grading_result_id

    async def list_graded_students(
        self,
        enterprise_id: UUID,
        exam_id: Optional[UUID] = None,
        limit: int = 10,
        offset: int = 0
    ) -> Tuple[List[Dict[str, Any]], int]:
        """
        Retrieve a paginated list of students and their overall grades.
        Checks each record's row checksum for database integrity, flagging tampered items.
        """
        query_params = [enterprise_id]
        where_clause = "WHERE enterprise_id = $1"

        if exam_id:
            query_params.append(exam_id)
            where_clause += " AND exam_id = $2"

        # Count query
        count_query = f"SELECT COUNT(*) FROM grading_results {where_clause}"
        total_count = await self._pool.fetchval(count_query, *query_params)

        if total_count == 0:
            return [], 0

        # Paginated query
        limit_idx = len(query_params) + 1
        offset_idx = len(query_params) + 2
        query_params.extend([limit, offset])

        data_query = f"""
            SELECT 
                id, session_id, exam_id, candidate_id, enrollment_id,
                total_max_points, total_awarded_points, percentage,
                status, graded_by, row_checksum, version, created_at, updated_at
            FROM grading_results
            {where_clause}
            ORDER BY created_at DESC
            LIMIT ${limit_idx} OFFSET ${offset_idx}
        """

        rows = await self._pool.fetch(data_query, *query_params)
        results = []

        for row in rows:
            record = dict(row)
            # Verify cryptographic integrity
            expected_checksum = calculate_row_checksum(
                session_id=str(row["session_id"]),
                candidate_id=str(row["candidate_id"]),
                total_max_points=float(row["total_max_points"]),
                total_awarded_points=float(row["total_awarded_points"]),
                percentage=float(row["percentage"]),
                version=row["version"]
            )
            record["is_tampered"] = (row["row_checksum"] != expected_checksum)
            
            # Parse graded_by into a GraderInfo dict
            raw_grader = record.get("graded_by", "system")
            if raw_grader.startswith("user:"):
                record["graded_by"] = {
                    "id": raw_grader[5:],
                    "type": "human"
                }
            else:
                record["graded_by"] = {
                    "id": raw_grader,
                    "type": "system"
                }
                
            if record["is_tampered"]:
                logger.error(
                    "DATABASE CORRUPTION DETECTED! Grading result row ID %s (Session %s) has invalid checksum.",
                    row["id"], row["session_id"]
                )
            results.append(record)

        return results, total_count

    async def get_by_session(self, session_id: UUID) -> Optional[Dict[str, Any]]:
        """
        Retrieve grade details and question details for a single exam session.
        Validates the HMAC checksum on retrieval.
        """
        row = await self._pool.fetchrow(
            "SELECT * FROM grading_results WHERE session_id = $1",
            session_id
        )
        if not row:
            return None

        record = dict(row)
        
        # Parse graded_by into a GraderInfo dict
        raw_grader = record.get("graded_by", "system")
        if raw_grader.startswith("user:"):
            record["graded_by"] = {
                "id": raw_grader[5:],
                "type": "human"
            }
        else:
            record["graded_by"] = {
                "id": raw_grader,
                "type": "system"
            }

        expected_checksum = calculate_row_checksum(
            session_id=str(row["session_id"]),
            candidate_id=str(row["candidate_id"]),
            total_max_points=float(row["total_max_points"]),
            total_awarded_points=float(row["total_awarded_points"]),
            percentage=float(row["percentage"]),
            version=row["version"]
        )
        record["is_tampered"] = (row["row_checksum"] != expected_checksum)

        # Get question details
        qr_rows = await self._pool.fetch(
            "SELECT question_id, session_question_id, question_type, title, content, candidate_answer, max_points, awarded_points, status FROM grading_question_results WHERE grading_result_id = $1",
            row["id"]
        )
        question_results = []
        for qr in qr_rows:
            qr_dict = dict(qr)
            if qr_dict.get("candidate_answer") and isinstance(qr_dict["candidate_answer"], str):
                try:
                    qr_dict["candidate_answer"] = json.loads(qr_dict["candidate_answer"])
                except json.JSONDecodeError:
                    pass
            question_results.append(qr_dict)
            
        record["question_results"] = question_results

        return record

    async def update_grade_manually(
        self,
        session_id: UUID,
        new_awarded_points: float,
        actor_id: UUID,
        actor_role: str,
        reason: str,
        ip_address: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Manually override a student's grade with strict HMAC updates and audit logs.
        """
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                row = await conn.fetchrow(
                    "SELECT id, candidate_id, total_max_points, total_awarded_points, percentage, row_checksum, version FROM grading_results WHERE session_id = $1",
                    session_id
                )
                if not row:
                    raise ValueError(f"Grading result for session {session_id} not found.")

                # Integrity check before letting anyone update it
                stored_checksum = row["row_checksum"]
                expected_checksum = calculate_row_checksum(
                    session_id=str(session_id),
                    candidate_id=str(row["candidate_id"]),
                    total_max_points=float(row["total_max_points"]),
                    total_awarded_points=float(row["total_awarded_points"]),
                    percentage=float(row["percentage"]),
                    version=row["version"]
                )
                if stored_checksum != expected_checksum:
                    raise DataTamperingError(session_id)

                max_pts = float(row["total_max_points"])
                new_pct = round((new_awarded_points / max_pts) * 100, 2) if max_pts > 0 else 0.0
                new_version = row["version"] + 1

                # Recalculate HMAC checksum
                new_checksum = calculate_row_checksum(
                    session_id=str(session_id),
                    candidate_id=str(row["candidate_id"]),
                    total_max_points=max_pts,
                    total_awarded_points=new_awarded_points,
                    percentage=new_pct,
                    version=new_version
                )

                # Set transaction context for audit logging trigger
                await self._set_audit_context(conn, actor_id, actor_role, ip_address, reason)

                result = await conn.execute(
                    """
                    UPDATE grading_results SET
                        total_awarded_points = $1,
                        percentage = $2,
                        status = $3,
                        graded_by = $4,
                        row_checksum = $5,
                        updated_at = NOW()
                    WHERE session_id = $6 AND version = $7
                    """,
                    new_awarded_points,
                    new_pct,
                    GradingStatus.reviewed.value,
                    f"user:{str(actor_id)}",
                    new_checksum,
                    session_id,
                    row["version"]
                )

                if result == "UPDATE 0":
                    raise RuntimeError("Optimistic locking conflict during grade edit. Please try again.")

                return {
                    "session_id": session_id,
                    "previous_score": float(row["total_awarded_points"]),
                    "new_score": new_awarded_points,
                    "new_percentage": new_pct,
                    "status": GradingStatus.reviewed.value
                }

    async def update_question_grade_manually(
        self,
        session_id: UUID,
        session_question_id: UUID,
        new_question_score: float,
        actor_id: UUID,
        actor_role: str,
        reason: str,
        ip_address: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Manually override a specific question's score and recalculate the overall grade with strict HMAC updates.
        """
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                row = await conn.fetchrow(
                    "SELECT id, candidate_id, total_max_points, total_awarded_points, percentage, row_checksum, version FROM grading_results WHERE session_id = $1",
                    session_id
                )
                if not row:
                    raise ValueError(f"Grading result for session {session_id} not found.")

                result_id = row['id']

                # Integrity check before letting anyone update it
                stored_checksum = row["row_checksum"]
                expected_checksum = calculate_row_checksum(
                    session_id=str(session_id),
                    candidate_id=str(row["candidate_id"]),
                    total_max_points=float(row["total_max_points"]),
                    total_awarded_points=float(row["total_awarded_points"]),
                    percentage=float(row["percentage"]),
                    version=row["version"]
                )
                if stored_checksum != expected_checksum:
                    raise DataTamperingError(session_id)

                # Fetch question result
                qr_row = await conn.fetchrow(
                    "SELECT id, max_points, awarded_points FROM grading_question_results WHERE grading_result_id = $1 AND session_question_id = $2",
                    result_id, session_question_id
                )
                if not qr_row:
                    raise ValueError(f"Question {session_question_id} not found for session {session_id}.")

                prev_question_score = float(qr_row["awarded_points"])
                prev_total_score = float(row["total_awarded_points"])

                # Calculate new total awarded points
                score_diff = new_question_score - prev_question_score
                new_total_awarded_points = prev_total_score + score_diff

                max_pts = float(row["total_max_points"])
                new_pct = round((new_total_awarded_points / max_pts) * 100, 2) if max_pts > 0 else 0.0
                new_version = row["version"] + 1

                # Recalculate HMAC checksum
                new_checksum = calculate_row_checksum(
                    session_id=str(session_id),
                    candidate_id=str(row["candidate_id"]),
                    total_max_points=max_pts,
                    total_awarded_points=new_total_awarded_points,
                    percentage=new_pct,
                    version=new_version
                )

                # Set transaction context for audit logging trigger
                await self._set_audit_context(conn, actor_id, actor_role, ip_address, reason)

                # Update question result
                await conn.execute(
                    """
                    UPDATE grading_question_results SET
                        awarded_points = $1,
                        status = 'human_review'
                    WHERE id = $2
                    """,
                    new_question_score,
                    qr_row["id"]
                )

                # Update overall result
                result = await conn.execute(
                    """
                    UPDATE grading_results SET
                        total_awarded_points = $1,
                        percentage = $2,
                        status = 'reviewed',
                        graded_by = $3,
                        row_checksum = $4,
                        updated_at = NOW()
                    WHERE id = $5 AND version = $6
                    """,
                    new_total_awarded_points,
                    new_pct,
                    f"user:{str(actor_id)}",
                    new_checksum,
                    result_id,
                    row["version"]
                )

                if result == "UPDATE 0":
                    raise RuntimeError("Optimistic locking conflict during grade edit. Please try again.")

                return {
                    "session_id": session_id,
                    "session_question_id": session_question_id,
                    "previous_question_score": prev_question_score,
                    "new_question_score": new_question_score,
                    "previous_total_score": prev_total_score,
                    "new_total_score": new_total_awarded_points,
                    "new_percentage": new_pct,
                    "status": "reviewed"
                }

    async def get_audit_logs(self, session_id: UUID) -> List[Dict[str, Any]]:
        """Retrieve edit logs/audit trails for a particular grading result."""
        grading_result_id = await self._pool.fetchval(
            "SELECT id FROM grading_results WHERE session_id = $1",
            session_id
        )
        if not grading_result_id:
            return []

        rows = await self._pool.fetch(
            """
            SELECT id, action, actor_id, actor_role, old_values, new_values, changed_fields, ip_address, reason, created_at
            FROM grading_audit_log
            WHERE grading_result_id = $1
            ORDER BY created_at DESC
            """,
            grading_result_id
        )

        result = []
        for row in rows:
            record = dict(row)
            for field in ("old_values", "new_values"):
                val = record.get(field)
                if isinstance(val, str):
                    record[field] = json.loads(val)
            result.append(record)
        return result

    async def create_pending_result(
        self,
        session_id: UUID,
        exam_id: UUID,
        candidate_id: UUID,
        enterprise_id: UUID,
        enrollment_id: UUID,
    ) -> None:
        """Insert a 'pending' placeholder row before grading begins.

        Idempotent: ON CONFLICT (session_id) DO NOTHING means re-delivered
        Kafka messages will not raise an error or overwrite a row that is
        already graded (e.g. from a prior successful run).

        The row checksum is computed for the zero-score values at version=1 so
        that save_grading_report can verify and then UPDATE the row cleanly.
        """
        checksum = calculate_row_checksum(
            session_id=str(session_id),
            candidate_id=str(candidate_id),
            total_max_points=0.0,
            total_awarded_points=0.0,
            percentage=0.0,
            version=1,
        )
        await self._pool.execute(
            """
            INSERT INTO grading_results (
                session_id, exam_id, candidate_id, enterprise_id, enrollment_id,
                total_max_points, total_awarded_points, percentage,
                status, graded_by, row_checksum, version
            ) VALUES ($1, $2, $3, $4, $5, 0, 0, 0, $6, 'system', $7, 1)
            ON CONFLICT (session_id) DO NOTHING
            """,
            session_id,
            exam_id,
            candidate_id,
            enterprise_id,
            enrollment_id,
            GradingStatus.pending.value,
            checksum,
        )
        logger.info("Pending grading placeholder created for session=%s", session_id)

    async def get_grading_status(self, session_id: UUID) -> Optional[Dict[str, Any]]:
        """Lightweight status query — returns only the fields needed for polling.

        Includes enterprise_id so the handler can enforce tenant isolation
        without issuing a second full-detail query.
        Returns None if no grading record exists yet.
        """
        row = await self._pool.fetchrow(
            """
            SELECT session_id, enterprise_id, status, graded_by, percentage, updated_at
            FROM grading_results
            WHERE session_id = $1
            """,
            session_id,
        )
        return dict(row) if row else None
