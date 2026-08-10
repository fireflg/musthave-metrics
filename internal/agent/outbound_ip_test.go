package agent

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboundIP_MatchesListenerFamily(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	ip := net.ParseIP(outboundIP(lis.Addr().String()))
	require.NotNil(t, ip)
	assert.NotNil(t, ip.To4(), "для IPv4-сервера агент обязан заявлять IPv4, иначе доверенная подсеть его отвергнет")
}

func TestOutboundIP_LocalhostResolvesToListenerFamily(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(t, err)

	ip := net.ParseIP(outboundIP("http://localhost:" + port))
	require.NotNil(t, ip)
	assert.NotNil(t, ip.To4())
}

func TestOutboundIP_UnreachableServerFallsBack(t *testing.T) {
	assert.NotPanics(t, func() { outboundIP("127.0.0.1:1") })
}

func TestLazyIP_ResolvesOnce(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	l := newLazyIP(lis.Addr().String())
	first := l.get()
	require.NotEmpty(t, first)
	assert.Equal(t, first, l.get())
}
