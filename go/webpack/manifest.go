package webpack

type HashType int

const (
	SHA256 HashType = iota
	SHA384
	SHA512
)

type Manifest struct {
	metadata    Metadata
	hashTypes   []HashType
	subpackages []string
}
