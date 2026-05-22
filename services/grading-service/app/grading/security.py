import hmac
import hashlib
from app.config import settings

def calculate_row_checksum(
    session_id: str,
    candidate_id: str,
    total_max_points: float,
    total_awarded_points: float,
    percentage: float,
    version: int
) -> str:
    """
    Generate an HMAC-SHA256 checksum to guarantee row-level data integrity
    and detect any out-of-band tampering directly in the database.
    """
    # Standardize values to form a deterministic message
    msg = (
        f"{str(session_id).lower()}:"
        f"{str(candidate_id).lower()}:"
        f"{float(total_max_points):.2f}:"
        f"{float(total_awarded_points):.2f}:"
        f"{float(percentage):.2f}:"
        f"{int(version)}"
    )
    key = settings.GRADING_SECRET_KEY.encode("utf-8")
    return hmac.new(key, msg.encode("utf-8"), hashlib.sha256).hexdigest()
