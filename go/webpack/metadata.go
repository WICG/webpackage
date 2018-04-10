package webpack

import (
	"net/url"
	"time"
)

type Metadata struct {
	Origin      *url.URL
	Date        time.Time
	OtherFields map[string]interface{}
}
