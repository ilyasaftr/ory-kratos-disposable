package domain

// OryWebhookResponse represents the response to send back to Ory Kratos
type OryWebhookResponse struct {
	Messages []MessageGroup `json:"messages,omitempty"`
}

// MessageGroup groups messages for a specific field
type MessageGroup struct {
	InstancePtr string    `json:"instance_ptr"`
	Messages    []Message `json:"messages"`
}

// Message represents a single validation message
type Message struct {
	ID      int                    `json:"id"`
	Text    string                 `json:"text"`
	Type    string                 `json:"type"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// NewErrorResponse creates an error response for disposable email
func NewErrorResponse(email, domain string) OryWebhookResponse {
	return OryWebhookResponse{
		Messages: []MessageGroup{
			{
				InstancePtr: "#/traits/email",
				Messages: []Message{
					{
						ID:   4000001,
						Text: "Disposable email addresses are not allowed",
						Type: "error",
						Context: map[string]interface{}{
							"email":  email,
							"domain": domain,
						},
					},
				},
			},
		},
	}
}
