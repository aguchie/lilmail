package utils

import (
	"lilmail/models"
	"sort"
	"strings"
	"time"
)

// ThreadContainer holds email threading information
type ThreadContainer struct {
	Message  *models.Email
	Parent   *ThreadContainer
	Children []*ThreadContainer
	Next     *ThreadContainer
}

// ThreadBuilder builds email threads using the JWZ algorithm
type ThreadBuilder struct {
	idTable map[string]*ThreadContainer
	rootSet []*ThreadContainer
}

// NewThreadBuilder creates a new thread builder
func NewThreadBuilder() *ThreadBuilder {
	return &ThreadBuilder{
		idTable: make(map[string]*ThreadContainer),
		rootSet: []*ThreadContainer{},
	}
}

// BuildThreads implements the JWZ threading algorithm
func (tb *ThreadBuilder) BuildThreads(emails []*models.Email) []*models.EmailThread {
	// Step 1: Create containers for all messages
	for _, email := range emails {
		tb.getContainer(email.MessageID).Message = email
		
		// Link References
		for i, ref := range email.References {
			parent := tb.getContainer(ref)
			child := tb.getContainer(email.References[min(i+1, len(email.References)-1)])
			
			if parent != child && child.Parent == nil {
				child.Parent = parent
				parent.Children = append(parent.Children, child)
			}
		}
		
		// Link In-Reply-To
		if email.InReplyTo != "" {
			parent := tb.getContainer(email.InReplyTo)
			child := tb.getContainer(email.MessageID)
			
			if parent != child && child.Parent == nil {
				child.Parent = parent
				parent.Children = append(parent.Children, child)
			}
		}
	}
	
	// Step 2: Find root set
	for _, container := range tb.idTable {
		if container.Parent == nil {
			tb.rootSet = append(tb.rootSet, container)
		}
	}
	
	// Step 3: Group into threads
	threads := tb.groupThreads()
	
	// Step 4: Sort threads by date (newest first)
	sort.Slice(threads, func(i, j int) bool {
		return threads[i].LastDate.After(threads[j].LastDate)
	})
	
	return threads
}

// getContainer retrieves or creates a container for a message ID
func (tb *ThreadBuilder) getContainer(messageID string) *ThreadContainer {
	if messageID == "" {
		return &ThreadContainer{}
	}
	
	container, exists := tb.idTable[messageID]
	if !exists {
		container = &ThreadContainer{}
		tb.idTable[messageID] = container
	}
	return container
}

// groupThreads groups containers into email threads
func (tb *ThreadBuilder) groupThreads() []*models.EmailThread {
	threads := []*models.EmailThread{}
	
	for _, root := range tb.rootSet {
		if root.Message == nil && len(root.Children) == 0 {
			continue
		}
		
		thread := &models.EmailThread{
			ID:       generateThreadID(root),
			Messages: []models.Email{},
		}
		
		// Collect all messages in the thread
		tb.collectMessages(root, thread)
		
		if len(thread.Messages) == 0 {
			continue
		}
		
		// Set thread properties
		thread.MessageCount = len(thread.Messages)
		thread.Subject = cleanSubject(thread.Messages[0].Subject)
		
		// Collect participants
		participants := make(map[string]bool)
		for _, msg := range thread.Messages {
			participants[msg.From] = true
			if msg.To != "" {
				for _, to := range strings.Split(msg.To, ",") {
					participants[strings.TrimSpace(to)] = true
				}
			}
		}
		thread.Participants = mapKeys(participants)
		
		// Find latest date
		for _, msg := range thread.Messages {
			if msg.Date.After(thread.LastDate) {
				thread.LastDate = msg.Date
			}
		}
		
		// Check for unread messages
		for _, msg := range thread.Messages {
			if !hasFlag(msg.Flags, "\\Seen") {
				thread.Unread = true
				break
			}
		}
		
		// Check for attachments
		for _, msg := range thread.Messages {
			if msg.HasAttachments {
				thread.HasAttachment = true
				break
			}
		}
		
		threads = append(threads, thread)
	}
	
	return threads
}

// collectMessages recursively collects all messages in a thread
func (tb *ThreadBuilder) collectMessages(container *ThreadContainer, thread *models.EmailThread) {
	if container.Message != nil {
		thread.Messages = append(thread.Messages, *container.Message)
	}
	
	for _, child := range container.Children {
		tb.collectMessages(child, thread)
	}
}

// cleanSubject removes Re:, Fwd:, etc. prefixes
func cleanSubject(subject string) string {
	subject = strings.TrimSpace(subject)
	prefixes := []string{"Re:", "RE:", "Fwd:", "FWD:", "Fw:"}
	
	for {
		cleaned := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(subject, prefix) {
				subject = strings.TrimSpace(subject[len(prefix):])
				cleaned = true
				break
			}
		}
		if !cleaned {
			break
		}
	}
	
	return subject
}

// generateThreadID generates a unique thread ID
func generateThreadID(root *ThreadContainer) string {
	if root.Message != nil && root.Message.MessageID != "" {
		return root.Message.MessageID
	}
	return time.Now().Format("20060102150405")
}

// hasFlag checks if a flag exists in the flags slice
func hasFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}

// mapKeys extracts keys from a map
func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
