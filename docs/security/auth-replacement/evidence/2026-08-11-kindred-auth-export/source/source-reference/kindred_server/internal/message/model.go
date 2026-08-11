// Package message provides the post-approval chat between a donor and the
// requester they approved. Conversations are scoped by request ID so each
// claim has its own thread.
//
// Messages are end-to-end encrypted with a v4 ratchet envelope. The server
// validates transport shape and sender identity signatures, and stores opaque ciphertext only; all ratchet
// state, skipped keys, and plaintext cache live on-device. Wire format:
// docs/E2E.md.
package message

import "time"

const (
	ProtocolVersionV4       = 4
	MaxSessionIDLen         = 128
	MaxCiphertextLen        = 4096
	MaxHeaderDataLen        = 4096
	MaxBootstrapIDLen       = 128
	MaxPreviewKeyIDLen      = 64
	MaxPreviewCiphertextLen = 1024
	MaxClientMessageIDLen   = 64

	hashB64Len                = 44
	signatureB64Len           = 88
	x25519PublicKeyB64Len     = 44
	mlkemCiphertextB64Len     = 1452
	nonceB64Len               = 16
	ratchetHeaderPQTypeNone   = "none"
	ratchetHeaderPQTypeHdr    = "hdr"
	ratchetHeaderPQTypeEK     = "ek"
	ratchetHeaderPQTypeEKACK  = "ekCt1Ack"
	ratchetHeaderPQTypeCT1ACK = "ct1Ack"
	ratchetHeaderPQTypeCT1    = "ct1"
	ratchetHeaderPQTypeCT2    = "ct2"
)

type Message struct {
	ID              string    `json:"id" dynamodbav:"id"`
	RequestID       string    `json:"requestId" dynamodbav:"requestId"`
	SenderID        string    `json:"senderId" dynamodbav:"senderId"`
	CreatedAt       time.Time `json:"createdAt" dynamodbav:"createdAt"`
	ClientMessageID string    `json:"clientMessageId,omitempty" dynamodbav:"clientMessageId,omitempty"`

	ProtocolVersion     int                  `json:"protocolVersion" dynamodbav:"protocolVersion"`
	SessionID           string               `json:"sessionId" dynamodbav:"sessionId"`
	RatchetHeader       RatchetHeader        `json:"ratchetHeader" dynamodbav:"ratchetHeader"`
	Nonce               string               `json:"nonce" dynamodbav:"nonce"`
	Ciphertext          string               `json:"ciphertext" dynamodbav:"ciphertext"`
	Bootstrap           *BootstrapData       `json:"bootstrap,omitempty" dynamodbav:"bootstrap,omitempty"`
	CiphertextHash      string               `json:"ciphertextHash" dynamodbav:"ciphertextHash"`
	PlaintextHash       string               `json:"plaintextHash" dynamodbav:"plaintextHash"`
	SenderSignature     string               `json:"senderSignature" dynamodbav:"senderSignature"`
	SenderKeyVersion    int                  `json:"senderKeyVersion" dynamodbav:"senderKeyVersion"`
	NotificationPreview *NotificationPreview `json:"notificationPreview,omitempty" dynamodbav:"notificationPreview,omitempty"`
}

type MessageReadState struct {
	RequestID                string    `json:"requestId"`
	UserID                   string    `json:"userId"`
	LastReadMessageID        string    `json:"lastReadMessageId"`
	LastReadMessageCreatedAt time.Time `json:"lastReadMessageCreatedAt"`
	LastReadAt               time.Time `json:"lastReadAt"`
}

type MarkConversationReadInput struct {
	LastReadMessageID        string    `json:"lastReadMessageId"`
	LastReadMessageCreatedAt time.Time `json:"lastReadMessageCreatedAt"`
}

type ConversationReadStates struct {
	Viewer       *MessageReadState
	Counterparty *MessageReadState
}

type PageDirection string

const (
	PageDirectionForward PageDirection = "forward"
	PageDirectionOlder   PageDirection = "older"
)

func ParsePageDirection(raw string) (PageDirection, bool) {
	switch raw {
	case "", string(PageDirectionForward):
		return PageDirectionForward, true
	case string(PageDirectionOlder):
		return PageDirectionOlder, true
	default:
		return "", false
	}
}

type SendInput struct {
	ClientMessageID     string               `json:"clientMessageId,omitempty"`
	ProtocolVersion     int                  `json:"protocolVersion"`
	SessionID           string               `json:"sessionId"`
	RatchetHeader       RatchetHeader        `json:"ratchetHeader"`
	Nonce               string               `json:"nonce"`
	Ciphertext          string               `json:"ciphertext"`
	Bootstrap           *BootstrapData       `json:"bootstrap,omitempty"`
	CiphertextHash      string               `json:"ciphertextHash"`
	PlaintextHash       string               `json:"plaintextHash"`
	SenderSignature     string               `json:"senderSignature"`
	SenderKeyVersion    int                  `json:"senderKeyVersion"`
	NotificationPreview *NotificationPreview `json:"notificationPreview,omitempty"`
}

type RatchetHeader struct {
	DHPublicKey         string           `json:"dhPublicKey" dynamodbav:"dhPublicKey"`
	PreviousChainLength int              `json:"previousChainLength" dynamodbav:"previousChainLength"`
	MessageNumber       int              `json:"messageNumber" dynamodbav:"messageNumber"`
	PQ                  *PQRatchetHeader `json:"pq,omitempty" dynamodbav:"pq,omitempty"`
}

type PQRatchetHeader struct {
	Epoch         int    `json:"epoch" dynamodbav:"epoch"`
	Type          string `json:"type" dynamodbav:"type"`
	MessageNumber int    `json:"messageNumber" dynamodbav:"messageNumber"`
	Data          string `json:"data,omitempty" dynamodbav:"data,omitempty"`
}

type BootstrapData struct {
	SignedX25519PrekeyID     string `json:"signedX25519PrekeyId" dynamodbav:"signedX25519PrekeyId"`
	OneTimeMLKEMPrekeyID     string `json:"oneTimeMLKEMPrekeyId" dynamodbav:"oneTimeMLKEMPrekeyId"`
	SenderEphemeralPublicKey string `json:"senderEphemeralPublicKey" dynamodbav:"senderEphemeralPublicKey"`
	MLKEMCiphertext          string `json:"mlkemCiphertext" dynamodbav:"mlkemCiphertext"`
}

type NotificationPreview struct {
	Version       int    `json:"version" dynamodbav:"version"`
	KeyID         string `json:"keyId" dynamodbav:"keyId"`
	KEMCiphertext string `json:"kemCiphertext" dynamodbav:"kemCiphertext"`
	Nonce         string `json:"nonce" dynamodbav:"nonce"`
	Ciphertext    string `json:"ciphertext" dynamodbav:"ciphertext"`
	Signature     string `json:"signature" dynamodbav:"signature"`
}
