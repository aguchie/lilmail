package api

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"lilmail/models"
	"lilmail/utils"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/emersion/go-imap"
)

// FetchMessages retrieves messages from a specified folder
func (c *Client) FetchMessages(folderName string, limit uint32) ([]models.Email, error) {
	mbox, err := c.client.Select(folderName, false)
	if err != nil {
		return nil, fmt.Errorf("error selecting folder %s: %v", folderName, err)
	}

	if mbox.Messages == 0 {
		return []models.Email{}, nil
	}

	from := uint32(1)
	if mbox.Messages > limit {
		from = mbox.Messages - limit + 1
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(from, mbox.Messages)

	messages := make(chan *imap.Message, limit)
	// Add header fetch for References
	// Add header fetch for References
	section := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Specifier: imap.HeaderSpecifier,
			Fields:    []string{"REFERENCES"},
		},
		Peek: true,
	}

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchBody,
		imap.FetchBodyStructure,
		imap.FetchUid,
		section.FetchItem(),
	}

	done := make(chan error, 1)
	go func() {
		done <- c.client.Fetch(seqSet, items, messages)
	}()

	var emails []models.Email
	for msg := range messages {
		email, err := c.processMessage(msg)
		if err != nil {
			fmt.Printf("Error processing message %d: %v\n", msg.Uid, err)
			continue
		}

		// Extract References header
		if r := msg.GetBody(section); r != nil {
			headerBytes, _ := ioutil.ReadAll(r)
			headerStr := string(headerBytes)
			// Parse "References: <...>"
			if idx := strings.Index(headerStr, ":"); idx > -1 {
				refs := strings.TrimSpace(headerStr[idx+1:])
				// Split by whitespace
				email.References = strings.Fields(refs)
			}
		}

		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return emails, fmt.Errorf("error during fetch: %v", err)
	}

	return emails, nil
}

// FetchThreads retrieves messages and organizes them into threads using JWZ algorithm
func (c *Client) FetchThreads(folderName string, limit uint32) ([]*models.EmailThread, error) {
	// First, fetch all messages
	emails, err := c.FetchMessages(folderName, limit)
	if err != nil {
		return nil, err
	}

	// Extract threading info from message headers
	for i := range emails {
		email := &emails[i]
		
		// Ensure MessageID exists
		if email.MessageID == "" {
			email.MessageID = email.ID // Fallback to UID
		}

		// References were populated in FetchMessages
		
		// In-Reply-To is part of Envelope, so it should be there, but verify processMessage logic
		if email.InReplyTo == "" && len(email.References) > 0 {
			// Some clients only send References. The last reference is usually the parent.
			email.InReplyTo = email.References[len(email.References)-1]
		}
	}

	// Build threads using JWZ algorithm
	threadBuilder := utils.NewThreadBuilder()
	threads := threadBuilder.BuildThreads(convertToEmailPointers(emails))

	return threads, nil
}

// Helper function to convert []models.Email to []*models.Email
func convertToEmailPointers(emails []models.Email) []*models.Email {
	result := make([]*models.Email, len(emails))
	for i := range emails {
		result[i] = &emails[i]
	}
	return result
}


// FetchMessagesPaginated retrieves messages with pagination support
func (c *Client) FetchMessagesPaginated(folderName string, page, pageSize uint32) (*models.PaginatedEmails, error) {
	mbox, err := c.client.Select(folderName, false)
	if err != nil {
		return nil, fmt.Errorf("error selecting folder %s: %v", folderName, err)
	}

	if mbox.Messages == 0 {
		return models.NewPaginatedEmails([]models.Email{}, page, pageSize, 0), nil
	}

	// Calculate message range for the requested page
	totalMessages := mbox.Messages
	totalPages := (totalMessages + pageSize - 1) / pageSize
	
	// Validate page number
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Calculate start and end indices (IMAP uses 1-based indexing, newest messages have higher indices)
	// We want to show newest messages first
	end := totalMessages - ((page - 1) * pageSize)
	start := end - pageSize + 1
	if start < 1 {
		start = 1
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(start, end)

	messages := make(chan *imap.Message, pageSize)
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchBody,
		imap.FetchBodyStructure,
		imap.FetchUid,
	}

	done := make(chan error, 1)
	go func() {
		done <- c.client.Fetch(seqSet, items, messages)
	}()

	var emails []models.Email
	for msg := range messages {
		email, err := c.processMessage(msg)
		if err != nil {
			fmt.Printf("Error processing message %d: %v\n", msg.Uid, err)
			continue
		}
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("error during fetch: %v", err)
	}

	// Reverse the emails array to show newest first
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return models.NewPaginatedEmails(emails, page, pageSize, totalMessages), nil
}

func (c *Client) FetchSingleMessage(folderName, uid string) (models.Email, error) {
	uidNum, err := strconv.ParseUint(uid, 10, 32)
	if err != nil {
		return models.Email{}, fmt.Errorf("invalid UID: %v", err)
	}

	// Select the folder
	_, err = c.client.Select(folderName, true)
	if err != nil {
		return models.Email{}, fmt.Errorf("error selecting folder %s: %v", folderName, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uint32(uidNum))

	// Define the items to fetch
	section := &imap.BodySectionName{
		Peek: true,
	}

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchBodyStructure,
		imap.FetchUid,
		section.FetchItem(),
	}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.client.UidFetch(seqSet, items, messages)
	}()

	var msg *imap.Message
	for m := range messages {
		msg = m
		break
	}

	if err := <-done; err != nil {
		return models.Email{}, fmt.Errorf("fetch error: %v", err)
	}

	if msg == nil {
		return models.Email{}, fmt.Errorf("message not found")
	}

	return c.processMessage(msg)
}

// DeleteMessage deletes a specific message by its UID
func (c *Client) DeleteMessage(folderName, uid string) error {
	uidNum, err := parseUID(uid)
	if err != nil {
		return fmt.Errorf("invalid UID: %v", err)
	}

	_, err = c.client.Select(folderName, false)
	if err != nil {
		return fmt.Errorf("error selecting folder %s: %v", folderName, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uidNum)

	// Mark as deleted
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}

	err = c.client.UidStore(seqSet, item, flags, nil)
	if err != nil {
		return fmt.Errorf("error marking message as deleted: %v", err)
	}

	// Expunge to permanently remove
	err = c.client.Expunge(nil)
	if err != nil {
		return fmt.Errorf("error expunging mailbox: %v", err)
	}

	return nil
}

// MarkMessageAsRead marks a message as read
func (c *Client) MarkMessageAsRead(folderName, uid string) error {
	return c.setMessageFlag(folderName, uid, imap.SeenFlag, true)
}

// MarkMessageAsUnread marks a message as unread
func (c *Client) MarkMessageAsUnread(folderName, uid string) error {
	return c.setMessageFlag(folderName, uid, imap.SeenFlag, false)
}

// setMessageFlag is a helper function to set or remove flags
func (c *Client) setMessageFlag(folderName, uid string, flag string, add bool) error {
	uidNum, err := parseUID(uid)
	if err != nil {
		return fmt.Errorf("invalid UID: %v", err)
	}

	_, err = c.client.Select(folderName, false)
	if err != nil {
		return fmt.Errorf("error selecting folder %s: %v", folderName, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uidNum)

	var operation imap.FlagsOp
	if add {
		operation = imap.AddFlags
	} else {
		operation = imap.RemoveFlags
	}

	item := imap.FormatFlagsOp(operation, true)
	flags := []interface{}{flag}

	err = c.client.UidStore(seqSet, item, flags, nil)
	if err != nil {
		return fmt.Errorf("error setting message flag: %v", err)
	}

	return nil
}

// MoveMessage moves a message from one folder to another
func (c *Client) MoveMessage(sourceFolder, targetFolder, uid string) error {
	uidNum, err := parseUID(uid)
	if err != nil {
		return fmt.Errorf("invalid UID: %v", err)
	}

	// Select source folder
	_, err = c.client.Select(sourceFolder, false)
	if err != nil {
		return fmt.Errorf("error selecting source folder %s: %v", sourceFolder, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uidNum)

	// Copy to target folder
	err = c.client.UidCopy(seqSet, targetFolder)
	if err != nil {
		return fmt.Errorf("error copying message to %s: %v", targetFolder, err)
	}

	// Mark as deleted in source folder
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}

	err = c.client.UidStore(seqSet, item, flags, nil)
	if err != nil {
		return fmt.Errorf("error marking message as deleted: %v", err)
	}

	// Expunge to permanently remove from source
	err = c.client.Expunge(nil)
	if err != nil {
		return fmt.Errorf("error expunging mailbox: %v", err)
	}

	return nil
}

// processAttachments extracts attachments from the message
func (c *Client) processAttachments(msg *imap.Message) ([]models.Attachment, error) {
	var attachments []models.Attachment

	var processAttachmentPart func(bs *imap.BodyStructure, partNum []int) error
	processAttachmentPart = func(bs *imap.BodyStructure, partNum []int) error {
		if bs == nil {
			return nil
		}

		isAttachment := bs.Disposition == "attachment" ||
			(bs.Disposition == "inline" && bs.MIMEType != "text")

		if isAttachment {
			section := &imap.BodySectionName{}
			if len(partNum) > 0 {
				section.Specifier = imap.PartSpecifier(strings.Join(strings.Fields(fmt.Sprint(partNum)), "."))
			}

			r := msg.GetBody(section)
			if r == nil {
				return fmt.Errorf("no body for attachment part %v", partNum)
			}

			content, err := io.ReadAll(r)
			if err != nil {
				return fmt.Errorf("error reading attachment content: %v", err)
			}

			attachment := models.Attachment{
				Filename:    bs.DispositionParams["filename"],
				ContentType: fmt.Sprintf("%s/%s", bs.MIMEType, bs.MIMESubType),
				Size:        len(content),
				Content:     content,
			}

			attachments = append(attachments, attachment)
		}

		for i, part := range bs.Parts {
			newPartNum := append(partNum, i+1)
			if err := processAttachmentPart(part, newPartNum); err != nil {
				return err
			}
		}

		return nil
	}

	err := processAttachmentPart(msg.BodyStructure, nil)
	return attachments, err
}

func (c *Client) processMessage(msg *imap.Message) (models.Email, error) {
	email := models.Email{
		ID:    fmt.Sprintf("%d", msg.Uid),
		Flags: msg.Flags,
	}

	// Process envelope information
	if msg.Envelope != nil {
		email.Subject = msg.Envelope.Subject
		email.Date = msg.Envelope.Date

		// Process From addresses
		if len(msg.Envelope.From) > 0 && msg.Envelope.From[0] != nil {
			email.From = msg.Envelope.From[0].Address()
			email.FromName = msg.Envelope.From[0].PersonalName
		}

		// Process To addresses
		if len(msg.Envelope.To) > 0 {
			var toAddresses []string
			var toNames []string
			for _, addr := range msg.Envelope.To {
				if addr != nil {
					toAddresses = append(toAddresses, addr.Address())
					if addr.PersonalName != "" {
						toNames = append(toNames, addr.PersonalName)
					}
				}
			}
			email.To = strings.Join(toAddresses, ", ")
			email.ToNames = toNames
		}

		// Process CC addresses
		if len(msg.Envelope.Cc) > 0 {
			var ccAddresses []string
			for _, addr := range msg.Envelope.Cc {
				if addr != nil {
					ccAddresses = append(ccAddresses, addr.Address())
				}
			}
			email.Cc = strings.Join(ccAddresses, ", ")
		}
	}

	// Process body
	// Process body
	var section imap.BodySectionName
	r := msg.GetBody(&section)
	if r != nil {
		// Read the body
		body, err := ioutil.ReadAll(r)
		if err != nil {
			return email, fmt.Errorf("error reading body: %v", err)
		}

		// Debug
		log.Printf("Initial body length: %d", len(body))

		// Parse the message
		m, err := mail.ReadMessage(bytes.NewReader(body))
		if err != nil {
			return email, fmt.Errorf("error parsing message: %v", err)
		}

		// Debug content type
		contentType := m.Header.Get("Content-Type")
		log.Printf("Content-Type: %s", contentType)

		// Handle multipart messages
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err == nil && strings.HasPrefix(mediaType, "multipart/") {
			mr := multipart.NewReader(m.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("Error getting next part: %v", err)
					continue
				}

				// Debug part content type
				log.Printf("Part Content-Type: %s", p.Header.Get("Content-Type"))

				// Read the part
				partData, err := ioutil.ReadAll(p)
				if err != nil {
					log.Printf("Error reading part: %v", err)
					continue
				}

				// Debug part length
				log.Printf("Part length: %d", len(partData))

				partType := p.Header.Get("Content-Type")
				switch {
				case strings.Contains(partType, "text/plain"):
					email.Body = string(partData)
					log.Printf("Found plain text: %d bytes", len(email.Body))
				case strings.Contains(partType, "text/html"):
					// Sanitize HTML to prevent XSS
					sanitized := utils.SanitizeHTML(string(partData))
					email.HTML = template.HTML(sanitized)
					log.Printf("Found HTML: %d bytes (sanitized)", len(string(email.HTML)))
				}
			}
		} else {
			// Handle non-multipart messages
			bodyData, err := ioutil.ReadAll(m.Body)
			if err == nil {
				email.Body = string(bodyData)
				log.Printf("Non-multipart body: %d bytes", len(email.Body))
			}
		}

		// Add preview after all content is processed
		if email.Body != "" {
			email.Preview = createPreview(email.Body)
		} else if email.HTML != "" {
			stripped := stripHTML(string(email.HTML))
			email.Preview = createPreview(stripped)
		}
	}

	// Debug final state
	log.Printf("Final state - Body: %d bytes, HTML: %d bytes, Preview: %d bytes",
		len(email.Body), len(string(email.HTML)), len(email.Preview))
	// Process attachments if needed
	attachments, err := c.processAttachments(msg)
	if err != nil {
		log.Printf("Warning: error processing attachments: %v", err)
	}
	email.Attachments = attachments
	email.HasAttachments = len(attachments) > 0

	return email, nil
}

// Simple HTML tag stripping
func stripHTML(html string) string {
	var builder strings.Builder
	inTag := false

	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			builder.WriteRune(r)
		}
	}

	return strings.TrimSpace(builder.String())
}

func cleanPlainTextBody(body string) string {
	body = strings.TrimSpace(body)
	lines := strings.Split(body, "\n")
	var cleanedLines []string

	headerPattern := regexp.MustCompile(`^(From|To|Subject|Date|Message-ID|MIME-Version|Content-Type|Content-Transfer-Encoding):`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !headerPattern.MatchString(line) {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

func createPreview(text string) string {
	// Normalize whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Trim to preview length
	if len(text) > 150 {
		// Try to break at a word boundary
		if idx := strings.LastIndex(text[:150], " "); idx > 0 {
			return text[:idx] + "..."
		}
		return text[:150] + "..."
	}
	return text
}

func html2text(htmlStr string) string {
	// Simple HTML to text conversion
	text := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"<p>", "\n",
		"</p>", "\n",
		"&nbsp;", " ",
	).Replace(htmlStr)

	// Remove remaining HTML tags
	text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")

	// Decode HTML entities
	text = html.UnescapeString(text)

	// Clean up whitespace
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}

// Clean up the getMessageBody method as well
func (c *Client) getMessageBody(msg *imap.Message, wantHTML bool) string {
	if msg.BodyStructure == nil {
		return ""
	}

	var findSection func(bs *imap.BodyStructure, partNum []int) (string, bool)
	findSection = func(bs *imap.BodyStructure, partNum []int) (string, bool) {
		if bs == nil {
			return "", false
		}

		isDesiredPart := strings.ToLower(bs.MIMEType) == "text" &&
			((wantHTML && strings.ToLower(bs.MIMESubType) == "html") ||
				(!wantHTML && strings.ToLower(bs.MIMESubType) == "plain"))

		if isDesiredPart {
			section := &imap.BodySectionName{}
			if len(partNum) > 0 {
				section.Specifier = imap.PartSpecifier(strings.Join(strings.Fields(fmt.Sprint(partNum)), "."))
			}

			r := msg.GetBody(section)
			if r == nil {
				return "", false
			}

			body, err := io.ReadAll(r)
			if err != nil {
				return "", false
			}

			return string(body), true
		}

		for i, part := range bs.Parts {
			newPartNum := append(partNum, i+1)
			if body, found := findSection(part, newPartNum); found {
				return body, true
			}
		}

		return "", false
	}

	body, _ := findSection(msg.BodyStructure, nil)
	return body
}

// FetchMessagesByUIDs retrieves messages for a specific list of UIDs
func (c *Client) FetchMessagesByUIDs(folderName string, uids []uint32) ([]models.Email, error) {
	if len(uids) == 0 {
		return []models.Email{}, nil
	}

	_, err := c.client.Select(folderName, false)
	if err != nil {
		return nil, fmt.Errorf("error selecting folder %s: %v", folderName, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	// Add header fetch for References (same as FetchMessages)
	// Add header fetch for References (same as FetchMessages)
	section := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Specifier: imap.HeaderSpecifier,
			Fields:    []string{"REFERENCES"},
		},
		Peek: true,
	}

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchBody,
		imap.FetchBodyStructure,
		imap.FetchUid,
		section.FetchItem(),
	}

	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	go func() {
		done <- c.client.UidFetch(seqSet, items, messages)
	}()

	var emails []models.Email
	for msg := range messages {
		email, err := c.processMessage(msg)
		if err != nil {
			fmt.Printf("Error processing message %d: %v\n", msg.Uid, err)
			continue
		}
		
		// Extract References header
		if r := msg.GetBody(section); r != nil {
			headerBytes, _ := ioutil.ReadAll(r)
			headerStr := string(headerBytes)
			if idx := strings.Index(headerStr, ":"); idx > -1 {
				refs := strings.TrimSpace(headerStr[idx+1:])
				email.References = strings.Fields(refs)
			}
		}

		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch error: %v", err)
	}

	// Reverse emails to show newest first for search results?
	// Usually search results are best shown by Date descending.
	// Since we fetched by UIDs, order might be arbitrary or ascending.
	// We'll rely on the client sort or sort here if needed.
	// Let's sort by Date Descending.
	// (Simple bubble sort for now or leave as is)
	// Leaving as is to minimize dependencies.
	
	return emails, nil
}
