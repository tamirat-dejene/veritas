from enum import Enum

class EventType(str, Enum):
    TAB_SWITCH = "tab_switch"
    MOUSE_INACTIVE = "mouse_inactive"
    FACE_NOT_DETECTED = "face_not_detected"
    MULTIPLE_FACES = "multiple_faces"
    IDENTITY_MISMATCH = "identity_mismatch"
    COPY_PASTE_ATTEMPT = "copy_paste_attempt"
    FULLSCREEN_EXIT = "fullscreen_exit"
    PERIODIC_FACE_OK = "periodic_face_ok"


class Severity(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


# Severity assigned to each event type
EVENT_SEVERITY: dict[str, str] = {
    EventType.TAB_SWITCH: Severity.MEDIUM,
    EventType.MOUSE_INACTIVE: Severity.LOW,
    EventType.FACE_NOT_DETECTED: Severity.HIGH,
    EventType.MULTIPLE_FACES: Severity.HIGH,
    EventType.IDENTITY_MISMATCH: Severity.CRITICAL,
    EventType.COPY_PASTE_ATTEMPT: Severity.MEDIUM,
    EventType.FULLSCREEN_EXIT: Severity.MEDIUM,
    EventType.PERIODIC_FACE_OK: Severity.LOW,
}

# Additive score weight per event occurrence.
# Positive = penalty, negative = recovery. Raw sum is clamped to [0, 100].
EVENT_SCORE_WEIGHT: dict[str, float] = {
    EventType.TAB_SWITCH: 5.0,
    EventType.MOUSE_INACTIVE: 3.0,
    EventType.FACE_NOT_DETECTED: 10.0,
    EventType.MULTIPLE_FACES: 15.0,
    EventType.IDENTITY_MISMATCH: 20.0,
    EventType.COPY_PASTE_ATTEMPT: 5.0,
    EventType.FULLSCREEN_EXIT: 4.0,
    # Negative weight = score recovery on good behavior
    EventType.PERIODIC_FACE_OK: -2.0,
}

# Maximum cumulative score contribution allowed per event type.
# Uncapped types (critical violations) are intentionally omitted.
EVENT_SCORE_CAPS: dict[str, float] = {
    EventType.TAB_SWITCH: 20.0,          # cap at 4 occurrences worth
    EventType.MOUSE_INACTIVE: 12.0,      # cap at 4 occurrences worth
    EventType.COPY_PASTE_ATTEMPT: 15.0,  # cap at 3 occurrences worth
    EventType.FULLSCREEN_EXIT: 16.0,     # cap at 4 occurrences worth
}

# Minimum seconds between two same-type events for the second to count
# toward the cheating score. Prevents rapid bursts from over-inflating.
COOLDOWN_WINDOW_SECONDS: float = 15.0
