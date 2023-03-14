package messages

import (
	"nathejk.dk/pkg/types"
)

type AttachmentUploaded struct {
	AttachmentID types.AttachmentID `json:"attachmentId"`
	URL          string             `json:"url"`
	Mimetype     string             `json:"mimetype"`
	Filename     string             `json:"filename"`
	Doctype      string             `json:"doctype"`
}
