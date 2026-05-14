-- +migrate Down

DROP TABLE IF EXISTS proctoring_session_scores;
DROP TABLE IF EXISTS proctoring_events;
DROP TYPE  IF EXISTS proctoring_severity;
