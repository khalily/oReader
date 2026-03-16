package opml

import "encoding/xml"

// opmlDocument represents the parsed OPML structure
type opmlDocument struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    opmlHead `xml:"head"`
	Body    opmlBody `xml:"body"`
}

// opmlHead represents the head section of OPML
// Note: DateCreated is omitempty for import parsing (where it may not exist)
type opmlHead struct {
	Title       string `xml:"title"`
	DateCreated string `xml:"dateCreated,omitempty"`
}

// opmlBody represents the body section of OPML
type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

// opmlOutline represents an outline element in OPML
type opmlOutline struct {
	Type     string        `xml:"type,attr,omitempty"`
	Text     string        `xml:"text,attr,omitempty"`
	Title    string        `xml:"title,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline"` // Nested outlines for import parsing
}
