package moonlight

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPendingPinRequests(t *testing.T) {
	manager := &PairingManager{}
	assert.Empty(t, manager.PendingPinRequests())

	firstRequest := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	manager.PendingPins.Store("newer", pendingPin{clientIP: "100.64.0.2", requestedAt: firstRequest.Add(time.Minute)})
	manager.PendingPins.Store("older", pendingPin{clientIP: "100.64.0.1", requestedAt: firstRequest})

	assert.Equal(t, []PendingPinRequest{
		{Secret: "older", ClientIP: "100.64.0.1", RequestedAt: firstRequest},
		{Secret: "newer", ClientIP: "100.64.0.2", RequestedAt: firstRequest.Add(time.Minute)},
	}, manager.PendingPinRequests())
}

func TestPostPin(t *testing.T) {
	manager := &PairingManager{}
	pins := make(chan PinSubmission, 1)
	manager.PendingPins.Store("secret", pendingPin{pins: pins})
	submission := PinSubmission{Pin: "1234", Username: "bwees"}

	require.NoError(t, manager.PostPin("secret", submission))
	assert.Equal(t, submission, <-pins)
	assert.Error(t, manager.PostPin("unknown", submission))
}
