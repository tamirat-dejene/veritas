-- Fix corrupted event_type values that were inserted as fully qualified enum strings
-- (e.g., 'EventType.IDENTITY_MISMATCH' instead of 'identity_mismatch')
-- due to a stringification bug in Python 3.11+.

UPDATE proctoring_events SET event_type = 'tab_switch' WHERE event_type = 'EventType.TAB_SWITCH';
UPDATE proctoring_events SET event_type = 'mouse_inactive' WHERE event_type = 'EventType.MOUSE_INACTIVE';
UPDATE proctoring_events SET event_type = 'face_not_detected' WHERE event_type = 'EventType.FACE_NOT_DETECTED';
UPDATE proctoring_events SET event_type = 'multiple_faces' WHERE event_type = 'EventType.MULTIPLE_FACES';
UPDATE proctoring_events SET event_type = 'identity_mismatch' WHERE event_type = 'EventType.IDENTITY_MISMATCH';
UPDATE proctoring_events SET event_type = 'copy_paste_attempt' WHERE event_type = 'EventType.COPY_PASTE_ATTEMPT';
UPDATE proctoring_events SET event_type = 'fullscreen_exit' WHERE event_type = 'EventType.FULLSCREEN_EXIT';
UPDATE proctoring_events SET event_type = 'periodic_face_ok' WHERE event_type = 'EventType.PERIODIC_FACE_OK';

-- Clean up any remaining corrupted data that didn't match the known patterns
DELETE FROM proctoring_events WHERE event_type LIKE 'EventType.%';
