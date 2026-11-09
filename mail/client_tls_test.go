package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestSendBindsTLSHandshakeToSMTPHost(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		useSSL bool
	}{
		{name: "STARTTLS"},
		{name: "implicit TLS", useSSL: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			certificateServer := httptest.NewTLSServer(nil)
			certificate := certificateServer.TLS.Certificates[0]
			certificateServer.Close()
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("Listen() error = %v", err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			type handshakeResult struct {
				name string
				err  error
			}
			results := make(chan handshakeResult, 1)
			go func() {
				connection, err := listener.Accept()
				if err != nil {
					return
				}
				defer connection.Close()
				if err := connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
					results <- handshakeResult{err: err}
					return
				}
				if !tt.useSSL {
					if err := smtpTLSGreeting(connection); err != nil {
						results <- handshakeResult{err: err}
						return
					}
				}
				var name string
				secure := tls.Server(connection, &tls.Config{
					Certificates: []tls.Certificate{certificate},
					MinVersion:   tls.VersionTLS12,
					GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
						name = hello.ServerName
						return nil, nil
					},
				})
				err = secure.HandshakeContext(t.Context())
				results <- handshakeResult{name: name, err: err}
			}()

			address, ok := listener.Addr().(*net.TCPAddr)
			if !ok {
				t.Fatalf("listener address type = %T, want *net.TCPAddr", listener.Addr())
			}
			client, err := NewClient(&ClientConfig{
				Host: "localhost", Port: address.Port, Username: "smoke", Password: "throwaway",
				UseSSL: tt.useSSL, MaxConns: 1, PoolWait: time.Second,
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			t.Cleanup(client.Close)
			message := NewMessage().SetFrom("sender@example.invalid").AddTo("recipient@example.invalid").
				SetSubject("TLS regression").SetHTML("<p>local test</p>")
			if err := client.Send(t.Context(), message); err == nil {
				t.Fatal("Send() accepted an untrusted SMTP certificate")
			}
			select {
			case result := <-results:
				if result.name != "localhost" {
					t.Fatalf("TLS ClientHello server name = %q, want localhost; handshake error = %v", result.name, result.err)
				}
				if result.err == nil {
					t.Fatal("TLS handshake trusted the test server certificate")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("SMTP peer did not observe a TLS handshake")
			}
		})
	}
}

func smtpTLSGreeting(connection net.Conn) error {
	protocol := textproto.NewConn(connection)
	if err := protocol.PrintfLine("220 localhost ESMTP"); err != nil {
		return err
	}
	line, err := protocol.ReadLine()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "EHLO ") {
		return fmt.Errorf("expected EHLO, got %q", line)
	}
	if err := protocol.PrintfLine("250-localhost\r\n250 STARTTLS"); err != nil {
		return err
	}
	line, err = protocol.ReadLine()
	if err != nil {
		return err
	}
	if line != "STARTTLS" {
		return fmt.Errorf("expected STARTTLS, got %q", line)
	}
	return protocol.PrintfLine("220 ready for TLS")
}
