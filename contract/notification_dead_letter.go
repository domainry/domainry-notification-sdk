package contract

// NotificationDeadLetter contains operational evidence only. Message content,
// recipient identities and source record identifiers are intentionally absent.
type NotificationDeadLetter struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	AttemptCount  int    `json:"attempt_count"`
	LastErrorCode string `json:"last_error_code"`
	UpdatedAt     string `json:"updated_at"`
}

type NotificationDeadLetterPage struct {
	Items      []NotificationDeadLetter `json:"items"`
	NextCursor string                   `json:"next_cursor,omitempty"`
}

type NotificationDeadLetterRedrive struct {
	ID       string `json:"id"`
	QueuedAt string `json:"queued_at"`
}
