package website

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Notifiarr/notifiarr/pkg/mnd"
	"golift.io/datacounter"
	"golift.io/version"
)

// httpClient is our custom http client to wrap Do and provide retries.
type httpClient struct {
	Retries int
	mnd.Logger
	*http.Client
}

func (s *Server) validAPIKey() error {
	if len(s.Config.Apps.APIKey) != APIKeyLength {
		return fmt.Errorf("%w: length must be %d characters", ErrInvalidAPIKey, APIKeyLength)
	}

	return nil
}

// unmarshalResponse attempts to turn the reply from the website into structured data.
func unmarshalResponse(url string, code int, body io.ReadCloser) (*Response, error) {
	var (
		buf     bytes.Buffer
		resp    Response
		counter = datacounter.NewReaderCounter(body)
	)

	defer func() {
		body.Close()
		mnd.Website.Add("POST Bytes Received", int64(counter.Count()))
	}()

	err := json.NewDecoder(io.TeeReader(counter, &buf)).Decode(&resp)
	if code < http.StatusOK || code > http.StatusIMUsed {
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %d %s (unmarshal error: %v), body: %s",
				ErrNon200, url, code, http.StatusText(code), err, buf.String())
		}

		return &resp, fmt.Errorf("%w: %s: %d %s",
			ErrNon200, url, code, http.StatusText(code))
	}

	if err != nil {
		return nil, fmt.Errorf("converting json response: %w, body: %s", err, buf.String())
	}

	return &resp, nil
}

// sendJSON posts a JSON payload to a URL. Returns the response body or an error.
func (s *Server) sendJSON(ctx context.Context, url string, data []byte, log bool) (int, io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return 0, nil, fmt.Errorf("creating http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.Config.Apps.APIKey)

	start := time.Now()

	resp, err := s.client.Do(req)
	if err != nil {
		s.debughttplog(nil, url, start, string(data), nil)
		return 0, nil, fmt.Errorf("making http request: %w", err)
	}

	if !s.Config.DebugEnabled() { // no debug, just return the body.
		return resp.StatusCode, resp.Body, nil
	}

	var buf bytes.Buffer
	tee := io.TeeReader(resp.Body, &buf)

	defer resp.Body.Close() // close this since we return a fake one after logging.

	if log {
		defer s.debughttplog(resp, url, start, string(data), tee)
	} else {
		defer s.debughttplog(resp, url, start, fmt.Sprintf("<data not logged, length:%d>", len(data)), tee)
	}

	return resp.StatusCode, io.NopCloser(&buf), nil
}

// Do performs an http Request with retries and logging!
func (h *httpClient) Do(req *http.Request) (*http.Response, error) { //nolint:cyclop
	req.Header.Set("User-Agent", fmt.Sprintf("%s v%s-%s %s", mnd.Title, version.Version, version.Revision, version.Branch))

	deadline, ok := req.Context().Deadline()
	if !ok {
		deadline = time.Now().Add(h.Timeout)
	}

	timeout := time.Until(deadline).Round(time.Millisecond)

	for retry := 0; ; retry++ {
		mnd.Website.Add(req.Method+" Requests", 1)

		resp, err := h.Client.Do(req)
		if err == nil {
			for i, c := range resp.Cookies() {
				h.ErrorfNoShare("Unexpected cookie [%v/%v] returned from website: %s", i+1, len(resp.Cookies()), c.String())
			}

			if resp.StatusCode < http.StatusInternalServerError &&
				(resp.StatusCode != http.StatusBadRequest || resp.Header.Get("content-type") != "text/html") {
				mnd.Website.Add(req.Method+" Bytes Sent", resp.Request.ContentLength)
				return resp, nil
			}

			// resp.StatusCode is 500 or higher, make that en error.
			// or resp.StatusCode is 400 and content-type is text/html (cloudflare error).
			size, _ := io.Copy(io.Discard, resp.Body) // must read the entire body when err == nil
			resp.Body.Close()                         // do not defer, because we're in a loop.
			mnd.Website.Add(req.Method+" Retries", 1)
			mnd.Website.Add(req.Method+" Bytes Sent", resp.Request.ContentLength)
			mnd.Website.Add(req.Method+" Bytes Received", size)
			// shoehorn a non-200 error into the empty http error.
			err = fmt.Errorf("%w: %s: %d bytes, %s", ErrNon200, req.URL, size, resp.Status)
		}

		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			if retry == 0 {
				return resp, fmt.Errorf("website req timed out after %s: %s: %w", timeout, req.URL, err)
			}

			return resp, fmt.Errorf("[%d/%d] website req timed out after %s, giving up: %w",
				retry+1, h.Retries+1, timeout, err)
		case retry == h.Retries:
			return resp, fmt.Errorf("[%d/%d] website req failed: %w", retry+1, h.Retries+1, err)
		default:
			h.ErrorfNoShare("[%d/%d] website req failed, retrying in %s, error: %v", retry+1, h.Retries+1, RetryDelay, err)
			time.Sleep(RetryDelay)
		}
	}
}

func (s *Server) debughttplog(resp *http.Response, url string, start time.Time, data string, body io.Reader) {
	headers := ""
	status := "0"

	if resp != nil {
		status = resp.Status

		for k, vs := range resp.Header {
			for _, v := range vs {
				headers += k + ": " + v + "\n"
			}
		}
	}

	if s.Config.Apps.MaxBody > 0 && len(data) > s.Config.Apps.MaxBody {
		data = fmt.Sprintf("%s <data truncated, max: %d>", data[:s.Config.Apps.MaxBody], s.Config.Apps.MaxBody)
	}

	if data == "" {
		s.Config.Debugf("Sent GET Request to %s in %s, Response (%s):\n%s\n%s",
			url, time.Since(start).Round(time.Microsecond), status,
			headers, readBodyForLog(body, int64(s.Config.Apps.MaxBody)))
	} else {
		s.Config.Debugf("Sent JSON Payload to %s in %s:\n%s\nResponse (%s):\n%s\n%s",
			url, time.Since(start).Round(time.Microsecond), data, status,
			headers, readBodyForLog(body, int64(s.Config.Apps.MaxBody)))
	}
}

// readBodyForLog truncates the response body, or not, for the debug log. errors are ignored.
func readBodyForLog(body io.Reader, max int64) string {
	if body == nil {
		return ""
	}

	if max > 0 {
		limitReader := io.LimitReader(body, max)
		bodyBytes, _ := io.ReadAll(limitReader)
		remaining, _ := io.Copy(io.Discard, body) // finish reading to the end.

		if remaining > 0 {
			return fmt.Sprintf("%s <body truncated, max: %d>", string(bodyBytes), max)
		}

		return string(bodyBytes)
	}

	bodyBytes, _ := io.ReadAll(body)

	return string(bodyBytes)
}

func (s *Server) watchSendDataChan(ctx context.Context) {
	defer func() {
		defer s.Config.CapturePanic()
		s.Config.Printf("==> Website notifier shutting down. No more ->website requests may be sent!")
	}()

	for data := range s.sendData {
		switch resp, elapsed, err := s.sendRequest(ctx, data); {
		case data.LogMsg == "", errors.Is(err, ErrInvalidAPIKey):
			continue
		case errors.Is(err, ErrNon200):
			s.Config.ErrorfNoShare("[%s requested] Sending (%v, buf=%d/%d): %s: %v%s",
				data.Event, elapsed, len(s.sendData), cap(s.sendData), data.LogMsg, err, resp)
		case err != nil:
			s.Config.Errorf("[%s requested] Sending (%v, buf=%d/%d): %s: %v%s",
				data.Event, elapsed, len(s.sendData), cap(s.sendData), data.LogMsg, err, resp)
		case !data.ErrorsOnly:
			s.Config.Printf("[%s requested] Sent (%v, buf=%d/%d): %s%s",
				data.Event, elapsed, len(s.sendData), cap(s.sendData), data.LogMsg, resp)
		default:
		}
	}

	close(s.stopSendData)
}

func (s *Server) sendRequest(ctx context.Context, data *Request) (*Response, time.Duration, error) {
	if err := s.validAPIKey(); err != nil {
		if data.respChan != nil {
			data.respChan <- &chResponse{
				Response: nil,
				Elapsed:  0,
				Error:    err,
			}
		}

		return nil, 0, err
	}

	var uri string

	if len(data.Params) > 0 {
		uri = data.Route.Path(data.Event, data.Params...)
	} else {
		uri = data.Route.Path(data.Event)
	}

	start := time.Now()
	resp, err := s.sendPayload(ctx, uri, data.Payload, data.LogPayload)
	elapsed := time.Since(start).Round(time.Millisecond)

	if data.respChan != nil {
		data.respChan <- &chResponse{
			Response: resp,
			Elapsed:  elapsed,
			Error:    err,
		}
	}

	return resp, elapsed, err
}
