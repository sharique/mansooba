// Package testparity records and replays a fixed HTTP request script against
// the API so behaviour can be compared before and after a framework change
// (feature 018, Echo v4 to v5). The recorded golden file is the contract: the
// same script, run against any build, must produce the same normalised
// results. See specs/018-echo-v5-mention-autocomplete/contracts/api-parity.md.
package testparity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Step is one request in the script. Variables written as {name} in Path,
// Body strings and Auth are replaced with values captured by earlier steps.
type Step struct {
	Name    string
	Method  string
	Path    string
	Auth    string            // variable holding a bearer token; empty = none
	Body    any               // JSON-encoded when non-nil
	Raw     []byte            // raw body, used instead of Body when non-nil
	Headers map[string]string // extra request headers
	Capture map[string]string // variable name -> dotted path in the JSON response
	Upload  *Upload           // multipart file upload
}

// Upload describes a multipart file part.
type Upload struct {
	Field, Filename, ContentType string
	Data                         []byte
}

// Result is the normalised outcome of one step.
type Result struct {
	Name     string            `json:"name"`
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     any               `json:"body,omitempty"`
	BodyText string            `json:"body_text,omitempty"`
}

var (
	reTime   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)
	reJWT    = regexp.MustCompile(`^eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
	reHex64  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reSigned = regexp.MustCompile(`X-Amz-Signature=`)
	reVar    = regexp.MustCompile(`\{([a-z_0-9]+)\}`)
	// reNumVar matches a whole JSON string that is only a variable, so a
	// captured number can be sent as a number: "{demo_id}" -> 7.
	reNumVar = regexp.MustCompile(`"\{([a-z_0-9]+)\}"`)
	reDigits = regexp.MustCompile(`^[0-9]+$`)
)

// headerKeys are the response headers that are part of the contract.
var headerKeys = []string{
	"Content-Type", "Cache-Control", "Vary", "Content-Disposition", "Location",
	"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials",
	"Access-Control-Allow-Methods", "Access-Control-Allow-Headers",
	"Access-Control-Expose-Headers", "Access-Control-Max-Age", "Allow",
	"X-Content-Type-Options", "WWW-Authenticate",
}

// Run executes steps against base and returns the normalised results.
func Run(base string, steps []Step) ([]Result, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Jar:     jar,
		Timeout: 60 * time.Second,
		// Do not follow redirects: a redirect is part of the contract.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	vars := map[string]string{}
	var out []Result
	for _, st := range steps {
		res, err := runStep(client, base, st, vars)
		if err != nil {
			return out, fmt.Errorf("step %q: %w", st.Name, err)
		}
		out = append(out, res)
	}
	return out, nil
}

func subst(s string, vars map[string]string) string {
	return reVar.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := vars[m[1:len(m)-1]]; ok {
			return v
		}
		return m
	})
}

func runStep(client *http.Client, base string, st Step, vars map[string]string) (Result, error) {
	var body io.Reader
	contentType := ""
	switch {
	case st.Upload != nil:
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, st.Upload.Field, st.Upload.Filename))
		h.Set("Content-Type", st.Upload.ContentType)
		part, err := mw.CreatePart(h)
		if err != nil {
			return Result{}, err
		}
		if _, err := part.Write(st.Upload.Data); err != nil {
			return Result{}, err
		}
		if err := mw.Close(); err != nil {
			return Result{}, err
		}
		body, contentType = &buf, mw.FormDataContentType()
	case st.Raw != nil:
		body = bytes.NewReader(st.Raw)
		contentType = "application/json"
	case st.Body != nil:
		b, err := json.Marshal(st.Body)
		if err != nil {
			return Result{}, err
		}
		js := reNumVar.ReplaceAllStringFunc(string(b), func(m string) string {
			name := m[2 : len(m)-2]
			if v, ok := vars[name]; ok && reDigits.MatchString(v) {
				return v
			}
			return m
		})
		body = strings.NewReader(subst(js, vars))
		contentType = "application/json"
	}
	req, err := http.NewRequest(st.Method, base+subst(st.Path, vars), body)
	if err != nil {
		return Result{}, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if st.Auth != "" {
		req.Header.Set("Authorization", "Bearer "+vars[st.Auth])
	}
	for k, v := range st.Headers {
		req.Header.Set(k, subst(v, vars))
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	res := Result{Name: st.Name, Status: resp.StatusCode, Headers: map[string]string{}}
	for _, k := range headerKeys {
		if v := resp.Header.Values(k); len(v) > 0 {
			res.Headers[k] = strings.Join(v, ", ")
		}
	}
	if cookies := resp.Header.Values("Set-Cookie"); len(cookies) > 0 {
		var norm []string
		for _, c := range cookies {
			norm = append(norm, normaliseCookie(c))
		}
		sort.Strings(norm)
		res.Headers["Set-Cookie"] = strings.Join(norm, " | ")
	}
	if len(res.Headers) == 0 {
		res.Headers = nil
	}

	if len(raw) > 0 {
		var parsed any
		if json.Unmarshal(raw, &parsed) == nil {
			if len(st.Capture) > 0 {
				for name, path := range st.Capture {
					if v, ok := lookup(parsed, path); ok {
						vars[name] = v
					}
				}
			}
			res.Body = normalise(parsed)
		} else {
			res.BodyText = string(raw)
			if len(res.BodyText) > 200 {
				// Long non-JSON bodies are third-party pages (the Swagger UI);
				// record that there was one, not its exact bytes.
				res.BodyText = "<text>"
			}
		}
	}
	return res, nil
}

func lookup(v any, path string) (string, bool) {
	for _, part := range strings.Split(path, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return "", false
		}
		v, ok = m[part]
		if !ok {
			return "", false
		}
	}
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return fmt.Sprintf("%d", int64(t)), true
	}
	return "", false
}

func normalise(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = normalise(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = normalise(x)
		}
		return t
	case string:
		switch {
		case reTime.MatchString(t):
			return "<time>"
		case reJWT.MatchString(t):
			return "<jwt>"
		case reHex64.MatchString(t):
			return "<token64>"
		case reSigned.MatchString(t):
			return "<presigned-url>"
		}
		return t
	}
	return v
}

var reCookieAttr = regexp.MustCompile(`^(?i)(expires|max-age)=.*$`)

func normaliseCookie(c string) string {
	parts := strings.Split(c, ";")
	name := strings.TrimSpace(parts[0])
	if i := strings.Index(name, "="); i >= 0 {
		name = name[:i] + "=<value>"
	}
	out := []string{name}
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if reCookieAttr.MatchString(p) {
			p = strings.ToLower(p[:strings.Index(p, "=")]) + "=<n>"
		}
		out = append(out, p)
	}
	return strings.Join(out, "; ")
}
