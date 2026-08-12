package scanner_test

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/scanner"
)

func TestClamAVScansTheExactByteStreamAndRecordsVersions(t *testing.T) {
	t.Parallel()

	daemon := newFakeClamD(t, fakeClamDConfig{
		version: "ClamAV 1.4.3/27411/Thu Jul 24 08:00:00 2026",
		result:  "stream: OK",
	})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	client, err := scanner.NewClamAV(scanner.ClamAVConfig{
		Address:             daemon.address,
		DialTimeout:         time.Second,
		MaximumSignatureAge: 48 * time.Hour,
		Clock:               func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewClamAV() error = %v", err)
	}

	body := []byte{0x00, 0x01, 0x02, '\n', 0xff, 'a', 'v', 'i', 'a'}
	result, err := client.Scan(context.Background(), strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if !result.Clean || result.Reason != "" {
		t.Fatalf("Scan() result = %+v", result)
	}
	if result.EngineVersion != "1.4.3" || result.SignatureVersion != "27411" {
		t.Fatalf("Scan() versions = engine %q, signatures %q", result.EngineVersion, result.SignatureVersion)
	}
	if !result.ScannedAt.Equal(now) {
		t.Fatalf("Scan() time = %s, want %s", result.ScannedAt, now)
	}
	if received := daemon.waitForStream(t); string(received) != string(body) {
		t.Fatalf("clamd received %v, want exact bytes %v", received, body)
	}
}

func TestClamAVReturnsTheDetectionSignatureWithoutTreatingItAsAnAdapterError(t *testing.T) {
	t.Parallel()

	daemon := newFakeClamD(t, fakeClamDConfig{
		version: "ClamAV 1.4.3/27411/Thu Jul 24 08:00:00 2026",
		result:  "stream: Win.Test.EICAR_HDB-1 FOUND",
	})
	client, err := scanner.NewClamAV(scanner.ClamAVConfig{
		Address:             daemon.address,
		DialTimeout:         time.Second,
		MaximumSignatureAge: 48 * time.Hour,
		Clock: func() time.Time {
			return time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("NewClamAV() error = %v", err)
	}

	result, err := client.Scan(context.Background(), strings.NewReader("EICAR"))
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Clean || result.Reason != "Win.Test.EICAR_HDB-1" {
		t.Fatalf("Scan() result = %+v", result)
	}
}

func TestClamAVFailsClosedOnDaemonErrorAndContextTimeout(t *testing.T) {
	t.Parallel()

	t.Run("daemon error", func(t *testing.T) {
		daemon := newFakeClamD(t, fakeClamDConfig{
			version: "ClamAV 1.4.3/27411/Thu Jul 24 08:00:00 2026",
			result:  "stream: INSTREAM size limit exceeded ERROR",
		})
		client, err := scanner.NewClamAV(scanner.ClamAVConfig{
			Address:             daemon.address,
			DialTimeout:         time.Second,
			MaximumSignatureAge: 48 * time.Hour,
			Clock: func() time.Time {
				return time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
			},
		})
		if err != nil {
			t.Fatalf("NewClamAV() error = %v", err)
		}
		if _, err := client.Scan(context.Background(), strings.NewReader("oversized")); err == nil ||
			!strings.Contains(err.Error(), "size limit") {
			t.Fatalf("Scan() daemon error = %v", err)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		daemon := newFakeClamD(t, fakeClamDConfig{
			version:     "ClamAV 1.4.3/27411/Thu Jul 24 08:00:00 2026",
			blockStream: true,
		})
		client, err := scanner.NewClamAV(scanner.ClamAVConfig{
			Address:             daemon.address,
			DialTimeout:         time.Second,
			MaximumSignatureAge: 48 * time.Hour,
			Clock: func() time.Time {
				return time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
			},
		})
		if err != nil {
			t.Fatalf("NewClamAV() error = %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if _, err := client.Scan(ctx, strings.NewReader("timeout")); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Scan() timeout error = %v", err)
		}
	})
}

func TestClamAVReadinessRejectsStaleSignatures(t *testing.T) {
	t.Parallel()

	daemon := newFakeClamD(t, fakeClamDConfig{
		version: "ClamAV 1.4.3/27411/Thu Jul 10 08:00:00 2026",
		result:  "stream: OK",
	})
	client, err := scanner.NewClamAV(scanner.ClamAVConfig{
		Address:             daemon.address,
		DialTimeout:         time.Second,
		MaximumSignatureAge: 48 * time.Hour,
		Clock: func() time.Time {
			return time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("NewClamAV() error = %v", err)
	}
	if err := client.Ready(context.Background()); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("Ready() stale signature error = %v", err)
	}
}

type fakeClamDConfig struct {
	version     string
	result      string
	blockStream bool
}

type fakeClamD struct {
	address string
	streams chan []byte
}

func newFakeClamD(t *testing.T, config fakeClamDConfig) *fakeClamD {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake clamd: %v", err)
	}
	daemon := &fakeClamD{address: listener.Addr().String(), streams: make(chan []byte, 4)}
	var connections sync.WaitGroup
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			connections.Add(1)
			go func() {
				defer connections.Done()
				defer connection.Close()
				reader := bufio.NewReader(connection)
				command, err := reader.ReadString(0)
				if err != nil {
					return
				}
				switch command {
				case "zPING\x00":
					_, _ = io.WriteString(connection, "PONG\x00")
				case "zVERSION\x00":
					_, _ = io.WriteString(connection, config.version+"\x00")
				case "zINSTREAM\x00":
					var stream []byte
					for {
						var size uint32
						if err := binary.Read(reader, binary.BigEndian, &size); err != nil {
							return
						}
						if size == 0 {
							break
						}
						chunk := make([]byte, int(size))
						if _, err := io.ReadFull(reader, chunk); err != nil {
							return
						}
						stream = append(stream, chunk...)
					}
					daemon.streams <- stream
					if config.blockStream {
						_, _ = io.Copy(io.Discard, connection)
						return
					}
					_, _ = io.WriteString(connection, config.result+"\x00")
				}
			}()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-stopped
		connections.Wait()
	})
	return daemon
}

func (daemon *fakeClamD) waitForStream(t *testing.T) []byte {
	t.Helper()
	select {
	case body := <-daemon.streams:
		return body
	case <-time.After(time.Second):
		t.Fatal("fake clamd did not receive a stream")
		return nil
	}
}
