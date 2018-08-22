package version

type Version string

const (
	Version1b1 Version = "1b1"
	Version1b2 Version = "1b2"
)

func Parse(str string) (Version, bool) {
	switch Version(str) {
	case Version1b1:
		return Version1b1, true
	case Version1b2:
		return Version1b2, true
	}
	return "", false
}
