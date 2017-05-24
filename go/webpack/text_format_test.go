package webpack

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseText(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	unsignedSingleFile, err := ParseText("testdata/unsigned_single_file.manifest")
	require.NoError(err)
	assert.Len(unsignedSingleFile.parts, 1, "Wrong number of parts.")

	index := unsignedSingleFile.parts[0]
	assert.Equal(*staticUrl("https://example.com/index.html"), *index.url)

	assert.Len(index.requestHeaders, 0)
	assert.Equal(200, index.status)
	assert.Equal(HTTPHeaders{
		httpHeader("content-type", "text/html"),
		httpHeader("expires", "Mon, 1 Jan 2018 01:00:00 GMT"),
	}, index.responseHeaders)

	content, err := index.Content()
	require.NoError(err)
	bytes, err := ioutil.ReadAll(content)
	require.NoError(err)
	assert.Equal(string(bytes), "I am example.com's index.html\n")
}

func TestParseTextVaryHeader(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	varyValid, err := ParseTextContent("testdata/", strings.NewReader(`[Content]
https://example.com/index.html
Allowed: value

200
Content-Type: text/html
Expires: Mon, 1 Jan 2018 01:00:00 GMT
Vary: allowed

content/example.com/index.html
`))
	require.NoError(err)
	require.Len(varyValid.parts, 1)

	index := varyValid.parts[0]
	assert.Equal(HTTPHeaders{
		httpHeader("allowed", "value"),
	}, index.requestHeaders)
}

func TestParseTextRequestHeaderNotInVary(t *testing.T) {
	_, err := ParseTextContent("testdata/", strings.NewReader(`[Content]
https://example.com/index.html
DisAllowed: value

200
Content-Type: text/html
Expires: Mon, 1 Jan 2018 01:00:00 GMT
Vary: allowed

content/example.com/index.html
`))
	assert.Error(t, err)
}

func staticUrl(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestWriteText(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	pack := Package{
		manifest: Manifest{},
		parts: []*PackPart{
			&PackPart{
				url:            staticUrl("https://example.com/index.html"),
				requestHeaders: HTTPHeaders{},
				status:         200,
				responseHeaders: HTTPHeaders{
					httpHeader("Content-Type", "text/html"),
					httpHeader("Expires", "Mon, 1 Jan 2018 01:00:00 GMT"),
				},
				content: []byte("I am example.com's index.html\n"),
			},
		},
	}

	dir, err := ioutil.TempDir("", "package")
	require.NoError(err)
	defer os.RemoveAll(dir)

	base := filepath.Join(dir, "unsigned_single_file")
	require.NoError(WriteTextTo(base, &pack))

	manifestContents, err := ioutil.ReadFile(filepath.Join(dir, "unsigned_single_file.manifest"))
	require.NoError(err)
	expectedManifestContents := strings.Replace(`[Content]
https://example.com/index.html

content-type: text/html
expires: Mon, 1 Jan 2018 01:00:00 GMT

https/example.com/index.html
`, "\n", "\r\n", -1)
	assert.Equal(expectedManifestContents, string(manifestContents))

	// Check that exactly the contained files were written out, to subdirectories of the manifest's basename.
	filenames := []string{}
	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			filenames = append(filenames, path[len(base)+1:])
		}
		return err
	})
	require.NoError(err)
	assert.Equal([]string{"https/example.com/index.html"}, filenames)
}
