package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultMarketplaceIndexMaxBytes int64 = 4 << 20

var ErrRemoteMarketplaceUnavailable = errors.New("remote marketplace index unavailable")

type RemoteMarketplaceAdapter struct {
	client   marketplaceHTTPClient
	maxBytes int64
}

type MarketplaceCatalogService struct {
	local  *LocalMarketplaceService
	remote *RemoteMarketplaceAdapter
}

type MarketplaceCatalogResult struct {
	Local       LocalMarketplaceCatalog `json:"local"`
	Remote      *MarketplaceIndex       `json:"remote,omitempty"`
	RemoteError string                  `json:"remoteError,omitempty"`
}

type RemoteMarketplaceError struct {
	URL        string
	StatusCode int
	Err        error
}

type marketplaceHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewRemoteMarketplaceAdapter(client marketplaceHTTPClient) *RemoteMarketplaceAdapter {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &RemoteMarketplaceAdapter{
		client:   client,
		maxBytes: defaultMarketplaceIndexMaxBytes,
	}
}

func NewMarketplaceCatalogService(local *LocalMarketplaceService, remote *RemoteMarketplaceAdapter) *MarketplaceCatalogService {
	if local == nil {
		local = NewLocalMarketplaceService(nil)
	}
	if remote == nil {
		remote = NewRemoteMarketplaceAdapter(nil)
	}
	return &MarketplaceCatalogService{local: local, remote: remote}
}

func (s *MarketplaceCatalogService) List(ctx context.Context, localRoot string, remoteIndexURL string) (MarketplaceCatalogResult, error) {
	local, err := s.local.List(localRoot)
	if err != nil {
		return MarketplaceCatalogResult{}, err
	}
	result := MarketplaceCatalogResult{Local: local}
	if strings.TrimSpace(remoteIndexURL) == "" {
		return result, nil
	}
	remote, err := s.remote.Fetch(ctx, remoteIndexURL)
	if err != nil {
		result.RemoteError = err.Error()
		return result, nil
	}
	result.Remote = &remote
	return result, nil
}

func (a *RemoteMarketplaceAdapter) Fetch(ctx context.Context, indexURL string) (MarketplaceIndex, error) {
	indexURL = strings.TrimSpace(indexURL)
	if err := validateRemoteMarketplaceURL(indexURL); err != nil {
		return MarketplaceIndex{}, newRemoteMarketplaceError(indexURL, 0, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return MarketplaceIndex{}, newRemoteMarketplaceError(indexURL, 0, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return MarketplaceIndex{}, newRemoteMarketplaceError(indexURL, 0, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return MarketplaceIndex{}, newRemoteMarketplaceError(indexURL, resp.StatusCode, fmt.Errorf("unexpected status %d", resp.StatusCode))
	}

	maxBytes := a.maxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMarketplaceIndexMaxBytes
	}
	var index MarketplaceIndex
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxBytes+1))
	if err := decoder.Decode(&index); err != nil {
		return MarketplaceIndex{}, newRemoteMarketplaceError(indexURL, resp.StatusCode, err)
	}
	if decoder.InputOffset() > maxBytes {
		return MarketplaceIndex{}, newRemoteMarketplaceError(indexURL, resp.StatusCode, fmt.Errorf("index exceeds %d bytes", maxBytes))
	}
	if err := ValidateMarketplaceIndex(index); err != nil {
		return MarketplaceIndex{}, newRemoteMarketplaceError(indexURL, resp.StatusCode, err)
	}
	return index, nil
}

func (e RemoteMarketplaceError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s: %s returned status %d: %v", ErrRemoteMarketplaceUnavailable, e.URL, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("%s: %s: %v", ErrRemoteMarketplaceUnavailable, e.URL, e.Err)
}

func (e RemoteMarketplaceError) Unwrap() error {
	return e.Err
}

func (e RemoteMarketplaceError) Is(target error) bool {
	return target == ErrRemoteMarketplaceUnavailable
}

func newRemoteMarketplaceError(indexURL string, statusCode int, err error) error {
	return RemoteMarketplaceError{
		URL:        indexURL,
		StatusCode: statusCode,
		Err:        err,
	}
}

func validateRemoteMarketplaceURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("index url is invalid: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("index url must be http or https")
	}
	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("index url host is required")
	}
	return nil
}
