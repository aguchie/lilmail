package api

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SMTPClient handles email sending
type SMTPClient struct {
	server   string
	port     int
	email    string
	password string
}

// AttachmentData represents a file attachment
type AttachmentData struct {
	Filename    string
	ContentType string
	Data        []byte
}

// NewSMTPClient creates a new SMTP client
func NewSMTPClient(server string, port int, email, password string) *SMTPClient {
	return &SMTPClient{
		server:   server,
		port:     port,
		email:    email,
		password: password,
	}
}

// SendMail sends an email using SMTP with support for HTML and Attachments
func (c *SMTPClient) SendMail(to, subject, body string, isHTML bool, attachments []AttachmentData) error {
	// Debug print
	fmt.Printf("Connecting to %s:%d as %s\n", c.server, c.port, c.email)

	// Connect to the server
	addr := fmt.Sprintf("%s:%d", c.server, c.port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}
	defer client.Close()

	// Send EHLO with domain from email
	domain := GetDomainFromEmail(c.email)
	if err := client.Hello(domain); err != nil {
		return fmt.Errorf("hello failed: %v", err)
	}

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName:         c.server,
		InsecureSkipVerify: true,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("starttls failed: %v", err)
	}

	username := GetUsernameFromEmail(c.email)
	// Authenticate after TLS
	auth := smtp.PlainAuth("", username, c.password, c.server)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("auth failed: %v", err)
	}

	// Set sender
	if err = client.Mail(c.email); err != nil {
		return fmt.Errorf("mail from failed: %v", err)
	}

	// Set recipient
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to failed: %v", err)
	}

	// Send the email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("data failed: %v", err)
	}
	
	// Construct Headers
	now := time.Now().Format(time.RFC1123Z)
	mixedBoundary := fmt.Sprintf("mixed-%s", generateBoundary())
	altBoundary := fmt.Sprintf("alt-%s", generateBoundary())

	headers := make(map[string]string)
	headers["Date"] = now
	headers["From"] = fmt.Sprintf("%s <%s>", username, c.email)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Message-ID"] = fmt.Sprintf("<%s@%s>", generateMessageID(), domain)

	if len(attachments) > 0 {
		headers["Content-Type"] = fmt.Sprintf("multipart/mixed; boundary=\"%s\"", mixedBoundary)
	} else if isHTML {
		headers["Content-Type"] = fmt.Sprintf("multipart/alternative; boundary=\"%s\"", altBoundary)
	} else {
		headers["Content-Type"] = "text/plain; charset=\"utf-8\""
	}

	// Write headers
	var headerBuf bytes.Buffer
	for k, v := range headers {
		headerBuf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	headerBuf.WriteString("\r\n")
	if _, err := writer.Write(headerBuf.Bytes()); err != nil {
		return err
	}

	// Write Body
	if len(attachments) > 0 {
		// Start mixed multipart
		fmt.Fprintf(writer, "--%s\r\n", mixedBoundary)
		
		if isHTML {
			// Nested alternative multipart
			fmt.Fprintf(writer, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)
			writeAlternativePart(writer, body, altBoundary)
			fmt.Fprintf(writer, "--%s--\r\n", altBoundary)
		} else {
			// Plain text part
			fmt.Fprintf(writer, "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s\r\n", body)
		}

		// Attachments
		for _, att := range attachments {
			fmt.Fprintf(writer, "--%s\r\n", mixedBoundary)
			fmt.Fprintf(writer, "Content-Type: %s; name=\"%s\"\r\n", att.ContentType, att.Filename)
			fmt.Fprintf(writer, "Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename)
			fmt.Fprintf(writer, "Content-Transfer-Encoding: base64\r\n\r\n")

			// Base64 encode
			b64 := base64.StdEncoding.EncodeToString(att.Data)
			// Split into lines of 76 chars
			for i := 0; i < len(b64); i += 76 {
				end := i + 76
				if end > len(b64) {
					end = len(b64)
				}
				fmt.Fprintf(writer, "%s\r\n", b64[i:end])
			}
		}
		fmt.Fprintf(writer, "--%s--\r\n", mixedBoundary)

	} else if isHTML {
		writeAlternativePart(writer, body, altBoundary)
		fmt.Fprintf(writer, "--%s--\r\n", altBoundary)
	} else {
		// Simple text
		if _, err := writer.Write([]byte(body)); err != nil {
			return err
		}
	}
	
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("data close failed: %v", err)
	}

	return client.Quit()
}

func writeAlternativePart(w io.Writer, body string, boundary string) {
	// Plain text version (stripped HTML or raw body)
	fmt.Fprintf(w, "--%s\r\n", boundary)
	fmt.Fprintf(w, "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	// Simple strip for plain text fallback
	plainText := stripHTMLTags(body) 
	fmt.Fprintf(w, "%s\r\n", plainText)

	// HTML version
	fmt.Fprintf(w, "--%s\r\n", boundary)
	fmt.Fprintf(w, "Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
	fmt.Fprintf(w, "%s\r\n", body)
}

func stripHTMLTags(html string) string {
    // Basic stripper
	return strings.ReplaceAll(strings.ReplaceAll(html, "<br>", "\n"), "<div>", "\n") 
}

func generateBoundary() string {
	return fmt.Sprintf("%x", rand.Int63())
}

// generateMessageID creates a unique Message-ID for the email
func generateMessageID() string {
	return fmt.Sprintf("%d.%d.%d",
		time.Now().UnixNano(),
		os.Getpid(),
		rand.Int63())
}

// Helper func to detect content type
func DetectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt": return "text/plain"
	case ".html": return "text/html"
	case ".jpg", ".jpeg": return "image/jpeg"
	case ".png": return "image/png"
	case ".pdf": return "application/pdf"
	case ".zip": return "application/zip"
	default: return "application/octet-stream"
	}
}
