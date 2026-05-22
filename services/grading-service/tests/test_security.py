"""
Unit tests for app.grading.security — HMAC row-checksum generation.
"""
from app.grading.security import calculate_row_checksum
from tests.conftest import SESSION_ID, CANDIDATE_ID


class TestCalculateRowChecksum:
    """Verify deterministic HMAC-SHA256 checksum generation."""

    def _default_checksum(self, **overrides):
        params = dict(
            session_id=SESSION_ID,
            candidate_id=CANDIDATE_ID,
            total_max_points=100.0,
            total_awarded_points=85.0,
            percentage=85.0,
            version=1,
        )
        params.update(overrides)
        return calculate_row_checksum(**params)

    # --- Determinism ---
    def test_same_inputs_produce_same_checksum(self):
        assert self._default_checksum() == self._default_checksum()

    def test_checksum_is_hex_string_of_64_chars(self):
        cs = self._default_checksum()
        assert isinstance(cs, str)
        assert len(cs) == 64
        int(cs, 16)  # should not raise

    # --- Sensitivity to each field ---
    def test_different_session_id_changes_checksum(self):
        other = self._default_checksum(session_id="aaaaaaaa-bbbb-cccc-dddd-ffffffffffff")
        assert other != self._default_checksum()

    def test_different_candidate_id_changes_checksum(self):
        other = self._default_checksum(candidate_id="aaaaaaaa-bbbb-cccc-dddd-ffffffffffff")
        assert other != self._default_checksum()

    def test_different_total_max_points_changes_checksum(self):
        other = self._default_checksum(total_max_points=200.0)
        assert other != self._default_checksum()

    def test_different_awarded_points_changes_checksum(self):
        other = self._default_checksum(total_awarded_points=90.0)
        assert other != self._default_checksum()

    def test_different_percentage_changes_checksum(self):
        other = self._default_checksum(percentage=90.0)
        assert other != self._default_checksum()

    def test_different_version_changes_checksum(self):
        other = self._default_checksum(version=2)
        assert other != self._default_checksum()

    # --- Edge cases ---
    def test_zero_scores(self):
        cs = self._default_checksum(
            total_max_points=0.0, total_awarded_points=0.0, percentage=0.0
        )
        assert isinstance(cs, str) and len(cs) == 64

    def test_case_insensitivity_of_uuid(self):
        upper = self._default_checksum(session_id=SESSION_ID.upper())
        lower = self._default_checksum(session_id=SESSION_ID.lower())
        assert upper == lower
