package studio

type ProviderSourceAsset struct {
	StorageKey string `json:"storage_key"`
	ID         string `json:"id"`
	SourceURL  string `json:"source_url"`
	PreviewURL string `json:"preview_url"`
	MimeType   string `json:"mime_type"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}
