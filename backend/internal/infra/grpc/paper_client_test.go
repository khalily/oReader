package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaperClient_InvalidAddress(t *testing.T) {
	// With non-blocking dial, the client is created even for invalid addresses.
	// Connection errors surface when actual RPC calls are made.
	client, err := NewPaperClient("invalid:99999", 5*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	// Clean up
	client.Close()
}

func TestPaperClientInterface(t *testing.T) {
	// Verify the interface is implemented
	var _ PaperConverterClient = (*paperClient)(nil)
}

func TestPaperClient_ContextCancelled(t *testing.T) {
	// When context is cancelled before RPC, should return error
	client, err := NewPaperClient("localhost:50051", 1*time.Second)
	if err != nil {
		// gRPC server not available, skip
		t.Skip("gRPC server not available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.Convert(ctx, nil, "test.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
