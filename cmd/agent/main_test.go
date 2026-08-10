package main

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReporter_HTTP(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	reporter, err := newReporter(&agent.Config{ServerURL: "http://127.0.0.1:8080"}, &key.PublicKey)
	require.NoError(t, err)

	assert.IsType(t, &agent.Reporter{}, reporter)
}

func TestNewReporter_GRPC(t *testing.T) {
	reporter, err := newReporter(&agent.Config{GRPCAddr: "127.0.0.1:3200"}, nil)
	require.NoError(t, err)
	require.IsType(t, &agent.GRPCReporter{}, reporter)

	assert.NoError(t, reporter.(*agent.GRPCReporter).Close())
}

func TestNewReporter_GRPCBadAddr(t *testing.T) {
	_, err := newReporter(&agent.Config{GRPCAddr: "\x00bad"}, nil)
	assert.Error(t, err)
}
