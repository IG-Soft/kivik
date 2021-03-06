package kivik

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"

	"github.com/go-kivik/kivik/v4/driver"
	"github.com/go-kivik/kivik/v4/internal/mock"
)

func TestAttachmentMarshalJSON(t *testing.T) {
	type tst struct {
		att      *Attachment
		expected string
		err      string
	}
	tests := testy.NewTable()
	tests.Add("foo.txt", tst{
		att: &Attachment{
			Content:     ioutil.NopCloser(strings.NewReader("test attachment\n")),
			Filename:    "foo.txt",
			ContentType: "text/plain",
		},
		expected: `{
			"content_type": "text/plain",
			"data": "dGVzdCBhdHRhY2htZW50Cg=="
		}`,
	})
	tests.Add("revpos", tst{
		att: &Attachment{
			Content:     ioutil.NopCloser(strings.NewReader("test attachment\n")),
			Filename:    "foo.txt",
			ContentType: "text/plain",
			RevPos:      3,
		},
		expected: `{
			"content_type": "text/plain",
			"data": "dGVzdCBhdHRhY2htZW50Cg==",
			"revpos": 3
		}`,
	})
	tests.Add("follows", tst{
		att: &Attachment{
			Content:     ioutil.NopCloser(strings.NewReader("test attachment\n")),
			Filename:    "foo.txt",
			Follows:     true,
			ContentType: "text/plain",
			RevPos:      3,
		},
		expected: `{
			"content_type": "text/plain",
			"follows": true,
			"revpos": 3
		}`,
	})
	tests.Add("read error", tst{
		att: &Attachment{
			Content:     ioutil.NopCloser(&errorReader{}),
			Filename:    "foo.txt",
			ContentType: "text/plain",
		},
		err: "json: error calling MarshalJSON for type *kivik.Attachment: errorReader",
	})
	tests.Add("stub", tst{
		att: &Attachment{
			Content:     ioutil.NopCloser(strings.NewReader("content")),
			Stub:        true,
			Filename:    "foo.txt",
			ContentType: "text/plain",
			Size:        7,
		},
		expected: `{
			"content_type": "text/plain",
			"length": 7,
			"stub": true
		}`,
	})

	tests.Run(t, func(t *testing.T, test tst) {
		result, err := json.Marshal(test.att)
		testy.Error(t, test.err, err)
		if d := diff.JSON([]byte(test.expected), result); d != nil {
			t.Error(d)
		}
	})
}

func TestAttachmentUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string

		body     string
		expected *Attachment
		err      string
	}{
		{
			name: "stub",
			input: `{
					"content_type": "text/plain",
					"stub": true
				}`,
			expected: &Attachment{
				ContentType: "text/plain",
				Stub:        true,
			},
		},
		{
			name: "simple",
			input: `{
					"content_type": "text/plain",
					"data": "dGVzdCBhdHRhY2htZW50Cg=="
				}`,
			body: "test attachment\n",
			expected: &Attachment{
				ContentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(Attachment)
			err := json.Unmarshal([]byte(test.input), result)
			testy.Error(t, test.err, err)
			var body []byte
			defer result.Content.Close() // nolint: errcheck
			body, err = ioutil.ReadAll(result.Content)
			if err != nil {
				t.Fatal(err)
			}
			result.Content = nil
			if d := diff.Text(test.body, string(body)); d != nil {
				t.Errorf("Unexpected body:\n%s", d)
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Errorf("Unexpected result:\n%s", d)
			}
		})
	}
}

func TestAttachmentsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string

		expected Attachments
		err      string
	}{
		{
			name:     "no attachments",
			input:    "{}",
			expected: Attachments{},
		},
		{
			name: "one attachment",
			input: `{
				"foo.txt": {
					"content_type": "text/plain",
					"data": "dGVzdCBhdHRhY2htZW50Cg=="
				}
			}`,
			expected: Attachments{
				"foo.txt": &Attachment{
					Filename:    "foo.txt",
					ContentType: "text/plain",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var att Attachments
			err := json.Unmarshal([]byte(test.input), &att)
			testy.Error(t, test.err, err)
			for _, v := range att {
				_ = v.Content.Close()
				v.Content = nil
			}
			if d := diff.Interface(test.expected, att); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestAttachmentsIteratorNext(t *testing.T) {
	tests := []struct {
		name     string
		iter     *AttachmentsIterator
		expected *Attachment
		status   int
		err      string
	}{
		{
			name: "error",
			iter: &AttachmentsIterator{
				atti: &mock.Attachments{
					NextFunc: func(_ *driver.Attachment) error {
						return &Error{HTTPStatus: http.StatusBadGateway, Err: errors.New("error")}
					},
				},
			},
			status: http.StatusBadGateway,
			err:    "error",
		},
		{
			name: "success",
			iter: &AttachmentsIterator{
				atti: &mock.Attachments{
					NextFunc: func(att *driver.Attachment) error {
						*att = driver.Attachment{
							Filename: "foo.txt",
						}
						return nil
					},
				},
			},
			expected: &Attachment{
				Filename: "foo.txt",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.iter.Next()
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
