package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAttachmentIDRequired        = errors.New("attachment id is required")
	ErrAttachmentMessageIDRequired = errors.New("attachment message id is required")
	ErrFileSizeInvalid             = errors.New("")
)

type Attachment struct {
	id          uuid.UUID
	messageID   uuid.UUID
	fileName    FileName
	fileSize    int32
	contentType ContentType
	url         AttachmentURL
	width       *int32
	height      *int32
	createdAt   time.Time
}

func (a *Attachment) ID() uuid.UUID            { return a.id }
func (a *Attachment) MessageID() uuid.UUID     { return a.messageID }
func (a *Attachment) FileName() FileName       { return a.fileName }
func (a *Attachment) FileSize() int32          { return a.fileSize }
func (a *Attachment) ContentType() ContentType { return a.contentType }
func (a *Attachment) URL() AttachmentURL       { return a.url }
func (a *Attachment) Width() *int32            { return a.width }
func (a *Attachment) Height() *int32           { return a.height }
func (a *Attachment) CreatedAt() time.Time     { return a.createdAt }

func NewAttachment(
	messageID uuid.UUID,
	fileName FileName,
	fileSize int32,
	contentType ContentType,
	url AttachmentURL,
	width, height *int32,
) (*Attachment, error) {
	if messageID == uuid.Nil {
		return nil, ErrAttachmentMessageIDRequired
	}
	if fileSize <= 0 {
		return nil, ErrFileSizeInvalid
	}

	return &Attachment{
		id:          uuid.Must(uuid.NewV7()),
		messageID:   messageID,
		fileName:    fileName,
		fileSize:    fileSize,
		contentType: contentType,
		url:         url,
		width:       width,
		height:      height,
		createdAt:   time.Now().UTC(),
	}, nil
}

func ReconstituteAttachment(
	id, messageID uuid.UUID,
	fileName string,
	fileSize int32,
	contentType string,
	url string,
	width, height *int32,
	createdAt time.Time,
) (*Attachment, error) {
	if id == uuid.Nil {
		return nil, ErrAttachmentIDRequired
	}
	if messageID == uuid.Nil {
		return nil, ErrAttachmentMessageIDRequired
	}

	fnVO, err := NewFileName(fileName)
	if err != nil {
		return nil, err
	}

	ctVO, err := NewContentType(contentType)
	if err != nil {
		return nil, err
	}

	urlVO, err := NewAttachmentURL(url)
	if err != nil {
		return nil, err
	}

	return &Attachment{
		id:          id,
		messageID:   messageID,
		fileName:    fnVO,
		fileSize:    fileSize,
		contentType: ctVO,
		url:         urlVO,
		width:       width,
		height:      height,
		createdAt:   createdAt,
	}, nil
}
