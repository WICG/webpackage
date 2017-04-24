package webpack

import (
	"net/url"
	"time"
)

type Metadata struct {
	origin url.URL
	date   time.Time
}
