package controllers

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChartSelectsStreamPods(t *testing.T) {
	label := fmt.Sprintf("%s: %q", StreamLabel, StreamLabelValue)

	for _, chartFile := range []string{
		"../../charts/direwolf-operator/templates/stream-service.yaml",
		"../../charts/direwolf-operator/values.yaml",
	} {
		content, err := os.ReadFile(chartFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), label, chartFile)
	}
}
