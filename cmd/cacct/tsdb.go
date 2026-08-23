package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ceems-dev/ceems/pkg/api/helper"
	"github.com/ceems-dev/ceems/pkg/api/models"
	"github.com/ceems-dev/ceems/pkg/tsdb"
	"github.com/prometheus/common/model"
)

var (
	queryMDMu    = sync.RWMutex{}
	queryInstant = sync.RWMutex{}
	queryMD      []queryMetadata
)

// queryMetadata contains metadata information for each TSDB series. We dump
// metadata.json file in the output directory containing fingerprint of each
// query and name CSV files after this fingerprint.
// This allows end users to programatically reads CSV files and their metadata
// and do data processing in their favorite tools like pandas, numpy, etc.
type queryMetadata struct {
	Fingerprint string       `json:"fingerprint"`
	Labels      model.Metric `json:"labels"`
}

// CacctSample is a custom sample that we use in exporting data in CSV, JSON formats.
type CacctSample struct {
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

// executeRangeQueries executes range queries and saves results to CSV files.
func executeRangeQueries(logger *slog.Logger, config *Config, units []models.Unit, outDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// New TSDB client
	client, err := tsdb.New(config.TSDB.Web.URL, config.TSDB.Web.HTTPClientConfig, slog.New(slog.DiscardHandler))
	if err != nil {
		logger.Error("Failed to create a new TSDB client", "err", err)

		return fmt.Errorf("failed to create tsdb API client: %w", err)
	}

	// Get absolute path of outDir
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for directory for saving CSV files: %w", err)
	}

	// Create outDir for saving CSV files
	err = os.MkdirAll(absOutDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create directory for saving CSV files: %w", err)
	}

	logger.Debug("Time series data will be saved", "dir", absOutDir)

	// Get TSDB settings
	settings := getTSDBSettings(ctx, config, client)

	// Start a wait group
	wg := sync.WaitGroup{}

	// Fetch time series of each metric in separate go routine
	for _, unit := range units {
		for _, q := range config.TSDB.RangeQueries {
			wg.Add(1)

			// Template data
			// LSF job arrays will have IDs like 300[1], 300[2], etc. Prometheus expects the
			// square brackets to be escaped or else it will ignore the label values. This is
			// due to the fact that it will use regex expression to match the label values.
			tmplData := map[string]any{
				"UUIDs":                   strings.ReplaceAll(strings.ReplaceAll(unit.UUID, "[", `\[`), "]", `\]`),
				"ScrapeInterval":          settings.ScrapeInterval,
				"ScrapeIntervalMilli":     settings.ScrapeInterval.Milliseconds(),
				"EvaluationInterval":      settings.EvaluationInterval,
				"EvaluationIntervalMilli": settings.EvaluationInterval.Milliseconds(),
				"RateInterval":            settings.RateInterval,
				"Range":                   time.Duration((unit.EndedAtTS - unit.StartedAtTS) * int64(time.Millisecond)),
			}

			// Build query
			query, err := queryBuilder(q.Name, q.Query, tmplData)
			if err != nil {
				logger.Error("Failed to build TSDB query", "query", q.Query, "err", err)
				wg.Done()

				continue
			}

			// Fetch metrics from TSDB and write to CSV files
			go fetchRangeData(ctx, q, query, unit.StartedAtTS, unit.EndedAtTS, absOutDir, client, &wg)
		}
	}

	// Wait for all routines
	wg.Wait()

	// Dump metadata.json for time series data
	if len(queryMD) > 0 {
		writeMetadata(logger, queryMD, absOutDir)

		fmt.Fprintln(os.Stderr, "time series data saved to directory", absOutDir)
	} else {
		return errors.New("no metadata found for range queries")
	}

	return nil
}

// fetchRangeData retrieves range query results from TSDB.
func fetchRangeData(ctx context.Context, q TSDBQuery, query string, start int64, end int64, outDir string, client *tsdb.Client, wg *sync.WaitGroup) {
	defer wg.Done()

	// Make a range query
	results, err := client.RangeQuery(ctx, query, time.UnixMilli(start), time.UnixMilli(end), 10*time.Second, time.Minute)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to fetch time series for query", query, "err:", err)

		return
	}

	var md []queryMetadata

	// Open metric file for each UUID and write data
	for _, result := range results {
		if _, ok := result.Metric["uuid"]; ok {
			// Get fingerprint
			fp := result.Metric.Fingerprint().String()

			// Get labels
			labels := result.Metric

			// Replace series name in labels with queryID
			// This is more readable one and also allows us
			// to protect Prometheus series names
			labels["metric"] = model.LabelValue(q.Name)
			if q.Title != "" {
				labels["metric"] = model.LabelValue(q.Title)
			}

			if q.Help != "" {
				labels["help"] = model.LabelValue(q.Help)
			}

			delete(labels, "__name__")

			// Strip port number, if exists, from instance and rename it to nodename
			labels["nodename"] = model.LabelValue(strings.Split(string(labels["instance"]), ":")[0])
			delete(labels, "instance")

			// Delete Prometheus specific labels
			delete(labels, "job")
			delete(labels, "hostname")
			delete(labels, "manager")
			delete(labels, "cgrouphostname")

			// Add metadata of query
			md = append(md, queryMetadata{
				Fingerprint: fp,
				Labels:      labels,
			})

			// Create file name based on fingerprint
			csvFilepath := filepath.Join(outDir, fp+".csv")

			writer, f, err := newCSVWriter(csvFilepath)
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to open file to write metrics:", err, "file:", csvFilepath)

				continue
			}

			defer f.Close()

			// Write header
			err = writer.Write([]string{"timestamp", "value"})
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to write header:", err, "file:", csvFilepath)
			}

			// Write records
			for _, value := range result.Values {
				err := writer.Write([]string{value.Timestamp.String(), value.Value.String()})
				if err != nil {
					fmt.Fprintln(os.Stderr, "failed to write data:", err, "file:", csvFilepath)
				}
			}

			// Flush writer
			writer.Flush()

			if writer.Error() != nil {
				fmt.Fprintln(os.Stderr, "failed to write data:", writer.Error(), "file:", csvFilepath)
			}
		}
	}

	// Append metadata to global var
	queryMDMu.Lock()
	defer queryMDMu.Unlock()

	queryMD = append(queryMD, md...)
}

// executeInstantQueries executes instant queries and returns map of query results.
func executeInstantQueries(logger *slog.Logger, config *Config, units []models.Unit) (map[string]map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// New TSDB client
	client, err := tsdb.New(config.TSDB.Web.URL, config.TSDB.Web.HTTPClientConfig, slog.New(slog.DiscardHandler))
	if err != nil {
		logger.Error("Failed to create a new TSDB client", "err", err)

		return nil, fmt.Errorf("failed to create tsdb API client: %w", err)
	}

	// Get slice of job IDs first for chunking
	uuids := make([]string, len(units))
	for iunit, unit := range units {
		uuids[iunit] = unit.UUID
	}

	// Get TSDB settings
	settings := getTSDBSettings(ctx, config, client)

	// Batch UUIDs into slices of 1000 so that we make TSDB requests for each 1000 units
	// This is to safeguard against OOM errors due to a very large number of units
	// that can spread across big time interval
	unitBatches := helper.ChunkBy(units, 1000)

	allInstantResults := make(map[string]map[string]string)

	// Initialise inner maps in allInstantResults
	for _, q := range config.TSDB.InstantQueries {
		allInstantResults[q.Name] = make(map[string]string)
	}

	// Fetch instant query results
	for _, unitBatch := range unitBatches {
		// Get UUIDs of the batch and min start and max end to compute range
		uuids := make([]string, len(unitBatch))
		minStartedTS := unitBatch[0].StartedAtTS

		maxEndedTS := unitBatch[0].EndedAtTS
		for iunit, unit := range unitBatch {
			uuids[iunit] = unit.UUID
			if unit.StartedAtTS < minStartedTS {
				minStartedTS = unit.StartedAtTS
			}

			if unit.EndedAtTS > maxEndedTS {
				maxEndedTS = unit.EndedAtTS
			}
		}
		// Start a wait group for each batch
		wg := sync.WaitGroup{}
		for _, q := range config.TSDB.InstantQueries {
			wg.Add(1)

			// Template data
			// LSF job arrays will have IDs like 300[1], 300[2], etc. Prometheus expects the
			// square brackets to be escaped or else it will ignore the label values. This is
			// due to the fact that it will use regex expression to match the label values.
			tmplData := map[string]any{
				"UUIDs":                   strings.ReplaceAll(strings.ReplaceAll(strings.Join(uuids, "|"), "[", `\[`), "]", `\]`),
				"ScrapeInterval":          settings.ScrapeInterval,
				"ScrapeIntervalMilli":     settings.ScrapeInterval.Milliseconds(),
				"EvaluationInterval":      settings.EvaluationInterval,
				"EvaluationIntervalMilli": settings.EvaluationInterval.Milliseconds(),
				"RateInterval":            settings.RateInterval,
				"Range":                   time.Duration((maxEndedTS - minStartedTS) * int64(time.Millisecond)),
			}

			// Build query
			query, err := queryBuilder(q.Name, q.Query, tmplData)
			if err != nil {
				logger.Error("Failed to build TSDB query", "query", q.Query, "err", err)
				wg.Done()

				continue
			}

			// Fetch instant query metrics from TSDB
			go fetchInstantData(ctx, q.Name, query, time.UnixMilli(maxEndedTS), allInstantResults, client, &wg)
		}

		// Wait for all routines
		wg.Wait()
	}

	return allInstantResults, nil
}

// fetchInstantData retrieves results of instant queries from TSDB.
func fetchInstantData(ctx context.Context, queryName string, query string, queryTime time.Time, allResults map[string]map[string]string, client *tsdb.Client, wg *sync.WaitGroup) {
	defer wg.Done()

	// Make a instant query
	results, err := client.Query(ctx, query, queryTime, time.Minute)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to fetch instant query results", query, "err:", err)

		return
	}

	// Append all the results to allResults maps
	queryResults := make(map[string][]CacctSample)

	for _, sample := range results {
		var (
			uuid  string
			entry CacctSample
		)

		// Intialise CacctSample.Labels maps
		entry.Labels = make(map[string]string)

		for ln, lv := range sample.Metric {
			if string(ln) == "uuid" {
				uuid = string(lv)

				continue
			}

			entry.Labels[string(ln)] = string(lv)
		}

		entry.Value = float64(sample.Value)
		queryResults[uuid] = append(queryResults[uuid], entry)
	}

	// Append current results to all results
	queryInstant.Lock()
	defer queryInstant.Unlock()

	for uuid, values := range queryResults {
		jsonString, err := json.Marshal(values)
		if err == nil {
			allResults[queryName][uuid] = string(jsonString)
		}
	}
}

// writeMetadata dumps the metadata.json file to outDir.
func writeMetadata(logger *slog.Logger, mds []queryMetadata, outDir string) {
	metadataFilepath := filepath.Join(outDir, "metadata.json")

	// Read existing metadata
	content, err := os.ReadFile(metadataFilepath)
	if err == nil {
		var existingMD []queryMetadata

		err := json.Unmarshal(content, &existingMD)
		if err == nil {
			mds = append(mds, existingMD...)
		}
	}

	// Dump metadata json
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetIndent("", "\t")

	err = encoder.Encode(mds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to encode metadata", "err:", err)

		return
	}

	file, err := os.OpenFile(metadataFilepath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create metadata.json file", "err:", err)

		return
	}

	defer file.Close()

	_, err = file.Write(buffer.Bytes())
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to write content to metadata.json file", "err:", err)

		return
	}

	logger.Debug("Metadata file saved", "file", metadataFilepath)
}

// getTSDBSettings return TSDB settings after overriding intervals from provided config.
func getTSDBSettings(ctx context.Context, config *Config, client *tsdb.Client) *tsdb.Settings {
	// Get current TSDB settings
	// Get rate and scrape intervals
	settings := client.Settings(ctx)

	// If scrape and evaluation intervals have been provided, use them instead of global value
	if config.TSDB.ScrapeInterval > 0 {
		settings.ScrapeInterval = time.Duration(config.TSDB.ScrapeInterval)
		settings.RateInterval = 4 * time.Duration(config.TSDB.ScrapeInterval)
	}

	if config.TSDB.EvaluationInterval > 0 {
		settings.EvaluationInterval = time.Duration(config.TSDB.EvaluationInterval)
	}

	return settings
}

// queryBuilder builds query from template and data.
func queryBuilder(name string, queryTemplate string, data map[string]any) (string, error) {
	tmpl := template.Must(template.New(name).Parse(queryTemplate))
	builder := &strings.Builder{}

	err := tmpl.Execute(builder, data)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

// newCSVWriter returns a new CSV writer.
func newCSVWriter(filename string) (*csv.Writer, *os.File, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, nil, err
	}

	// New instance of CSV writer
	writer := csv.NewWriter(f)

	return writer, f, nil
}
