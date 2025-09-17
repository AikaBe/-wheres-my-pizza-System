package postgre

import "errors"

var (
	ErrWorkerAlreadyActive      = errors.New("worker with this name is already active")
	ErrWorkerNotFound           = errors.New("worker not found")
	ErrWorkerRegistrationFailed = errors.New("worker registration failed")
)