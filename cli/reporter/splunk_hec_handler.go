// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
)

// splunkHecRegex recognizes a Splunk HTTP Event Collector endpoint. Splunk
// exposes the collector at /services/collector and /services/collector/event;
// both take the JSON envelope this handler sends.
var splunkHecRegex = regexp.MustCompile(`^https?://[^/]+/services/collector(/event)?/?$`)

// Splunk HEC settings that do not belong on the command line: the token is a
// credential, and the rest are deployment details.
const (
	splunkTokenEnv    = "SPLUNK_HEC_TOKEN"
	splunkIndexEnv    = "SPLUNK_HEC_INDEX"
	splunkSourceEnv   = "SPLUNK_HEC_SOURCE"
	splunkInsecureEnv = "SPLUNK_HEC_INSECURE"

	// splunkMaxBatchBytes keeps a request under Splunk's default 1 MB
	// max_content_length, leaving room for the last event to overshoot.
	splunkMaxBatchBytes = 512 * 1024

	splunkAttempts = 4
	splunkBackoff  = 2 * time.Second
)

// splunkHecHandler posts OCSF events to a Splunk HTTP Event Collector.
//
// Each event is wrapped in the HEC envelope rather than sent bare, so Splunk
// indexes it with the event's own timestamp and a sourcetype naming its OCSF
// class (`ocsf:compliance_finding`). Those are the sourcetypes the OCSF-CIM
// Add-On for Splunk keys off, which is what makes the events searchable as
// findings instead of as opaque JSON.
type splunkHecHandler struct {
	url     string
	conf    *PrintConfig
	client  *http.Client
	backoff time.Duration
}

func newSplunkHecHandler(url string, conf *PrintConfig) (*splunkHecHandler, error) {
	if conf.format != FormatOcsfJson {
		return nil, errors.Newf(
			"the Splunk HEC target takes OCSF events, please use --output ocsf-json (got %q)",
			formatName(conf.format))
	}

	client := &http.Client{Timeout: 60 * time.Second}
	if os.Getenv(splunkInsecureEnv) == "true" {
		log.Warn().Msg(splunkInsecureEnv + "=true, the Splunk server's TLS certificate will not be verified")
		client.Transport = &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint: gosec // opt-in, for self-signed HEC endpoints
		}
	}

	return &splunkHecHandler{url: url, conf: conf, client: client, backoff: splunkBackoff}, nil
}

func (h *splunkHecHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	token := os.Getenv(splunkTokenEnv)
	if token == "" {
		return errors.New("the Splunk HEC target needs a token, please set " + splunkTokenEnv)
	}

	events, err := ConvertToOCSF(report, h.conf)
	if err != nil {
		return err
	}
	if events.Len() == 0 {
		log.Warn().Msg("the scan produced no OCSF events, nothing sent to Splunk")
		return nil
	}

	batch := &bytes.Buffer{}
	sent := 0
	flush := func() error {
		if batch.Len() == 0 {
			return nil
		}
		if err := h.post(ctx, token, batch.Bytes()); err != nil {
			return err
		}
		batch.Reset()
		return nil
	}

	err = events.EachJSON(func(class string, event []byte) error {
		envelope, err := splunkEnvelope(class, event)
		if err != nil {
			return err
		}
		// Flush before adding, so a batch never exceeds the limit by more than
		// one event.
		if batch.Len()+len(envelope) > splunkMaxBatchBytes {
			if err := flush(); err != nil {
				return err
			}
		}
		batch.Write(envelope)
		sent++
		return nil
	})
	if err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}

	log.Info().Str("url", h.url).Int("events", sent).Msg("sent OCSF events to Splunk")
	return nil
}

// splunkEnvelope wraps one OCSF event in the HEC record format. HEC reads the
// indexed timestamp from `time` in epoch seconds and the rest of the fields as
// indexing metadata; the OCSF event itself is the payload of `event`.
func splunkEnvelope(class string, event []byte) ([]byte, error) {
	var meta struct {
		Time      int64                   `json:"time"`
		Device    *struct{ Name string }  `json:"device"`
		Resources []struct{ Name string } `json:"resources"`
	}
	// The envelope needs the event time and the asset; a payload we cannot read
	// is still sent, just without them.
	_ = json.Unmarshal(event, &meta)

	envelope := map[string]any{
		"sourcetype": "ocsf:" + class,
		"source":     envOr(splunkSourceEnv, "cnspec"),
		"event":      json.RawMessage(event),
	}
	if meta.Time > 0 {
		// OCSF timestamps are milliseconds, HEC wants seconds.
		envelope["time"] = float64(meta.Time) / 1000
	}
	if host := splunkHost(meta.Device, meta.Resources); host != "" {
		envelope["host"] = host
	}
	if index := os.Getenv(splunkIndexEnv); index != "" {
		envelope["index"] = index
	}

	raw, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	// HEC accepts concatenated JSON objects; the newline keeps them readable.
	return append(raw, '\n'), nil
}

func splunkHost(device *struct{ Name string }, resources []struct{ Name string }) string {
	if device != nil && device.Name != "" {
		return device.Name
	}
	if len(resources) > 0 {
		return resources[0].Name
	}
	return ""
}

// post sends one batch, retrying the failures that are worth retrying: Splunk
// being busy or restarting, and transport errors. A rejected payload is not
// retried, because it will be rejected again.
func (h *splunkHecHandler) post(ctx context.Context, token string, body []byte) error {
	var lastErr error
	for attempt := range splunkAttempts {
		if attempt > 0 {
			delay := h.backoff << (attempt - 1)
			log.Debug().Int("attempt", attempt+1).Dur("delay", delay).Msg("retrying the Splunk HEC request")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Splunk "+token)
		req.Header.Set("Content-Type", "application/json")

		res, err := h.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		payload, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		_ = res.Body.Close()

		switch {
		case res.StatusCode >= 200 && res.StatusCode < 300:
			return nil
		case res.StatusCode == http.StatusTooManyRequests, res.StatusCode >= 500:
			lastErr = errors.Newf("splunk returned %s: %s", res.Status, bytes.TrimSpace(payload))
		default:
			// 400 (bad payload), 401/403 (bad token), 404 (collector disabled)
			return errors.Newf("splunk rejected the events with %s: %s", res.Status, bytes.TrimSpace(payload))
		}
	}
	return errors.Wrapf(lastErr, "failed to send events to Splunk after %d attempts", splunkAttempts)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// formatName renders a Format for an error message.
func formatName(f Format) string {
	for name, format := range Formats {
		if format == f && name != "" {
			return name
		}
	}
	return strconv.Itoa(int(f))
}
