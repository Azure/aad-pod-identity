package framework

import (
	"time"
)

const (
	// CreateTimeout represents the duration it waits until a create operation times out.
	CreateTimeout = 10 * time.Second

	// CreatePolling represents the polling interval for a create operation.
	CreatePolling = 1 * time.Second

	// DeleteTimeout represents the duration it waits until a delete operation times out.
	DeleteTimeout = 1 * time.Minute

	// DeletePolling represents the polling interval for a delete operation.
	DeletePolling = 5 * time.Second

	// ListTimeout represents the duration it waits until a list operation times out.
	ListTimeout = 10 * time.Second

	// ListPolling represents the polling interval for a list operation.
	ListPolling = 1 * time.Second

	// GetTimeout represents the duration it waits until a get operation times out.
	GetTimeout = 1 * time.Minute

	// GetPolling represents the polling interval for a get operation.
	GetPolling = 5 * time.Second

	// UpdateTimeout represents the duration it waits until an update operation times out.
	UpdateTimeout = 10 * time.Second

	// UpdatePolling represents the polling interval for an update operation.
	UpdatePolling = 1 * time.Second

	// WaitTimeout represents the duration it waits until a wait operation times out.
	WaitTimeout = 5 * time.Minute

	// WaitPolling represents the polling interval for a wait operation.
	WaitPolling = 5 * time.Second
)
