package trace

import (
	"math"
	"sync"
	"testing"

	url_helpers "gitlab.com/gitlab-org/gitlab-runner/helpers/url"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariablesMasking(t *testing.T) {
	traceMessage := "This is the secret message cont@ining :secret duplicateValues ffixx"
	maskedValues := []string{
		"is",
		"duplicateValue",
		"duplicateValue",
		":secret",
		"cont@ining",
		"fix",
	}

	buffer, err := New()
	require.NoError(t, err)
	defer buffer.Close()

	buffer.SetMasked(maskedValues)

	_, err = buffer.Write([]byte(traceMessage))
	require.NoError(t, err)

	buffer.Finish()

	content, err := buffer.Bytes(0, 1000)
	require.NoError(t, err)

	assert.Equal(t, "Th[MASKED] [MASKED] the secret message [MASKED] [MASKED] [MASKED]s f[MASKED]x", string(content))
}

func TestTraceLimit(t *testing.T) {
	traceMessage := "This is the long message"

	buffer, err := New()
	require.NoError(t, err)
	defer buffer.Close()

	buffer.SetLimit(10)
	assert.Equal(t, 0, buffer.Size())

	for i := 0; i < 100; i++ {
		_, err = buffer.Write([]byte(traceMessage))
		require.NoError(t, err)
	}

	buffer.Finish()

	content, err := buffer.Bytes(0, 1000)
	require.NoError(t, err)

	expectedContent := "This is th\n\x1b[31;1mJob's log exceeded limit of 10 bytes.\x1b[0;m\n"
	assert.Equal(t, len(expectedContent), buffer.Size(), "unexpected buffer size")
	assert.Equal(t, "crc32:597f1ee1", buffer.Checksum())
	assert.Equal(t, expectedContent, string(content))
}

func TestDelayedMask(t *testing.T) {
	buffer, err := New()
	require.NoError(t, err)
	defer buffer.Close()

	buffer.Write([]byte("data before mask\n"))
	buffer.SetMasked([]string{"mask_me"})
	buffer.Write([]byte("data mask_me masked\n"))

	buffer.Finish()

	content, err := buffer.Bytes(0, 1000)
	require.NoError(t, err)

	expectedContent := "data before mask\ndata [MASKED] masked\n"
	assert.Equal(t, len(expectedContent), buffer.Size(), "unexpected buffer size")
	assert.Equal(t, "crc32:690f62e1", buffer.Checksum())
	assert.Equal(t, expectedContent, string(content))
}

func TestDelayedLimit(t *testing.T) {
	buffer, err := New()
	require.NoError(t, err)
	defer buffer.Close()

	buffer.Write([]byte("data before limit\n"))
	buffer.SetLimit(20)
	buffer.Write([]byte("data after limit\n"))

	buffer.Finish()

	content, err := buffer.Bytes(0, 1000)
	require.NoError(t, err)

	expectedContent := "data before limit\nda\n\x1b[31;1mJob's log exceeded limit of 20 bytes.\x1b[0;m\n"
	assert.Equal(t, len(expectedContent), buffer.Size(), "unexpected buffer size")
	assert.Equal(t, "crc32:faa63b66", buffer.Checksum())
	assert.Equal(t, expectedContent, string(content))
}

func TestTraceRace(t *testing.T) {
	buffer, err := New()
	require.NoError(t, err)
	defer buffer.Close()

	buffer.SetLimit(1000)

	load := []func(){
		func() { _, _ = buffer.Write([]byte("x")) },
		func() { buffer.SetMasked([]string{"x"}) },
		func() { buffer.SetLimit(1000) },
		func() { buffer.Checksum() },
	}

	var wg sync.WaitGroup
	for _, fn := range load {
		wg.Add(1)
		go func(fn func()) {
			defer wg.Done()

			for i := 0; i < 100; i++ {
				fn()
			}
		}(fn)
	}

	wg.Wait()

	buffer.Finish()

	_, err = buffer.Bytes(0, 1000)
	require.NoError(t, err)
}

const logLineStr = "hello world, this is a lengthy log line including secrets such as 'hello', and " +
	"https://example.com/?rss_token=foo&rss_token=bar and http://example.com/?authenticity_token=deadbeef and " +
	"https://example.com/?rss_token=foobar. it's longer than most log lines, but probably a good test for " +
	"anything that's benchmarking how fast it is to write log lines."

var logLineByte = []byte(logLineStr)

func BenchmarkBuffer10k(b *testing.B) {
	for i := 0; i < b.N; i++ {
		func() {
			buffer, err := New()
			require.NoError(b, err)
			defer buffer.Close()

			buffer.SetLimit(math.MaxInt64)
			buffer.SetMasked([]string{"hello"})

			const N = 10000
			b.ReportAllocs()
			b.SetBytes(int64(len(logLineByte) * N))
			for i := 0; i < N; i++ {
				_, _ = buffer.Write(logLineByte)
			}
			buffer.Finish()
		}()
	}
}

func BenchmarkBuffer10kWithURLScrub(b *testing.B) {
	for i := 0; i < b.N; i++ {
		func() {
			buffer, err := New()
			require.NoError(b, err)
			defer buffer.Close()

			buffer.SetLimit(math.MaxInt64)
			buffer.SetMasked([]string{"hello"})

			const N = 10000
			b.ReportAllocs()
			b.SetBytes(int64(len(logLineByte) * N))
			for i := 0; i < N; i++ {
				_, _ = buffer.Write([]byte(url_helpers.ScrubSecrets(logLineStr)))
			}
			buffer.Finish()
		}()
	}
}
