package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ceems-dev/ceems/pkg/api/models"
	http_config "github.com/prometheus/common/config"
)

const (
	headerKeyAcceptEncoding = "Accept-Encoding"
)

// stats returns units and usage structs by making requests to CEEMS API server.
func stats(
	logger *slog.Logger,
	config *Config,
	currentUser string,
	start time.Time,
	end time.Time,
	accounts []string,
	jobs []string,
	userNames []string,
	fields []string,
	includeSummaryStats bool,
) ([]models.Unit, []models.Usage, error) {
	// Add user header to HTTP config
	userHeaders := http_config.Header{
		Values: []string{currentUser},
	}
	if config.API.Web.HTTPClientConfig.HTTPHeaders != nil {
		config.API.Web.HTTPClientConfig.HTTPHeaders.Headers[config.API.UserHeaderName] = userHeaders
	} else {
		config.API.Web.HTTPClientConfig.HTTPHeaders = &http_config.Headers{
			Headers: map[string]http_config.Header{
				config.API.UserHeaderName: userHeaders,
			},
		}
	}

	// Add encoding header
	config.API.Web.HTTPClientConfig.HTTPHeaders.Headers[headerKeyAcceptEncoding] = http_config.Header{
		Values: []string{"gzip"},
	}

	// Parse web URL of API server
	apiURL, err := url.Parse(config.API.Web.URL)
	if err != nil {
		logger.Error("Failed to parse CEEMS API server URL", "err", errors.Unwrap(err))

		return nil, nil, fmt.Errorf("invalid API server url: %w", errors.Unwrap(err))
	}

	// Make a API server client from config file
	apiClient, err := http_config.NewClientFromConfig(config.API.Web.HTTPClientConfig, "ceems_api_server")
	if err != nil {
		logger.Error("Failed to create client for API server", "err", errors.Unwrap(err))

		return nil, nil, fmt.Errorf("failed to create client: %w", errors.Unwrap(err))
	}

	// Make url query parameters
	urlValues := url.Values{
		"cluster_id": []string{config.API.ClusterID},
		"from":       []string{strconv.FormatInt(start.Unix(), 10)},
		"to":         []string{strconv.FormatInt(end.Unix(), 10)},
		"field":      fields,
	}

	for _, account := range accounts {
		urlValues.Add("project", account)
	}

	for _, job := range jobs {
		urlValues.Add("uuid", job)
	}

	// Make request URL based on runtime config
	// Even if normal user make a request by requesting
	// with --user flag, if that user is not in admin list, empty
	// result will be returned
	var unitsReqURL, usageReqURL *url.URL

	if len(userNames) > 0 {
		// If --user flag does not contain special value "all", add them to the query.
		// If "all" is provided in the --user flag, dont add any query parameters so
		// that API server returns jobs of all users.
		if !slices.Contains(userNames, "all") {
			for _, user := range userNames {
				urlValues.Add("user", user)
			}
		}

		unitsReqURL = apiURL.JoinPath("/api/v1/units/admin")
		usageReqURL = apiURL.JoinPath("/api/v1/usage/current/admin")
	} else {
		unitsReqURL = apiURL.JoinPath("/api/v1/units")
		usageReqURL = apiURL.JoinPath("/api/v1/usage/current")
	}

	logger.Debug("Request to fetch units", "url", unitsReqURL.Redacted(), "units_query", urlValues.Encode())

	// If CEEMS URL is available make a API request
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Get all units in the given period
	units, err := doRequest[models.Unit](ctx, unitsReqURL, urlValues, apiClient)
	if err != nil {
		logger.Error("Failed to fetch units data from CEEMS API server", "err", err)

		return nil, nil, fmt.Errorf("failed to fetch jobs: %w", err)
	}

	// If summary stats are requested, include them
	var usage []models.Usage

	if includeSummaryStats {
		// Get all units in the given period
		// Always add user field as we will need to get total usage
		urlValues.Add("field", "username")
		urlValues.Add("field", "num_units")

		// If elapsed is requested we need to get total_time_seconds from usage API resource
		if slices.Contains(urlValues["field"], "elapsed") {
			urlValues.Add("field", "total_time_seconds")
		}

		logger.Debug("Request to fetch usage", "url", usageReqURL.Redacted(), "usage_query", urlValues.Encode())

		usage, err = doRequest[models.Usage](ctx, usageReqURL, urlValues, apiClient)
		if err != nil {
			logger.Error("Failed to fetch usage data from CEEMS API server", "err", err)

			return nil, nil, fmt.Errorf("failed to fetch usage: %w", err)
		}
	}

	return units, usage, nil
}

// doRequest does an API request to CEEMS API server and returns response.
func doRequest[T any](ctx context.Context, reqURL *url.URL, urlValues url.Values, client *http.Client) ([]T, error) {
	// Make a new request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Add role query parameter to request
	req.URL.RawQuery = urlValues.Encode()

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()

		// Any status code other than 200 should be treated as check failure
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			return nil, errNoPerm
		}

		// Read response body
		body, err := getBodyBytes(resp)
		if err != nil {
			return nil, err
		}

		// Unpack into data
		var data Response[T]

		err = json.Unmarshal(body, &data)
		if err != nil {
			return nil, err
		}

		// Check if Status is error
		if data.Status == "error" {
			return nil, errInternal
		}

		return data.Data, nil
	}
}

func getBodyBytes(res *http.Response) ([]byte, error) {
	if strings.EqualFold(res.Header.Get("Content-Encoding"), "gzip") {
		reader, err := gzip.NewReader(res.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		return io.ReadAll(reader)
	}

	return io.ReadAll(res.Body)
}
