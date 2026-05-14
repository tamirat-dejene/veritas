class ProctoringError(Exception):
    """Base class for all proctoring domain errors."""


class FaceNotRegisteredError(ProctoringError):
    """Raised when no face reference exists for the session."""


class NoClearFaceError(ProctoringError):
    """Raised when no usable face is detected in the submitted image."""


class SessionNotFoundError(ProctoringError):
    """Raised when candidate-service returns no session for the given ID."""


class InternalServiceError(ProctoringError):
    """Raised on downstream HTTP call failures."""
