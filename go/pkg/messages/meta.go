package messages

import (
	"net/http"
	"strings"

	"nathejk.dk/pkg/types"
)

type MetaID struct {
	ID string `json:"id"`
}

type ByUserMeta struct {
	UserID types.UserID `json:"userId"`
}

type MetadataRequestHeaders map[string]string

func (h *MetadataRequestHeaders) Set(header http.Header) {
	mrh := MetadataRequestHeaders{}
	for key, values := range header {
		mrh[key] = strings.Join(values, "\n")
	}
	*h = mrh
}

type Metadata struct {
	Producer       string                 `json:"producer"`
	Phase          string                 `json:"phase,omitempty"`
	RequestHeaders MetadataRequestHeaders `json:"requestHeaders,omitempty"`
}
