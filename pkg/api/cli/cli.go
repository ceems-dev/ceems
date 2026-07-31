//go:build cgo

// Package cli implements the CLI of the CEEMS API server app
package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/ceems-dev/ceems/internal/common"
	internal_runtime "github.com/ceems-dev/ceems/internal/runtime"
	"github.com/ceems-dev/ceems/internal/security"
	"github.com/ceems-dev/ceems/pkg/api/base"
	ceems_db "github.com/ceems-dev/ceems/pkg/api/db"
	ceems_http "github.com/ceems-dev/ceems/pkg/api/http"
	"github.com/ceems-dev/ceems/pkg/api/resource"
	"github.com/ceems-dev/ceems/pkg/api/updater"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"kernel.org/pub/linux/libs/security/libcap/cap"
)

// CEEMSAPIAppConfig contains the configuration of CEEMS API server.
type CEEMSAPIAppConfig struct {
	Server CEEMSAPIServerConfig `yaml:"ceems_api_server"`
}

// SetDirectory joins any relative file paths with dir.
func (c *CEEMSAPIAppConfig) SetDirectory(dir string) {
	c.Server.Admin.SetDirectory(dir)
}

// Validate validates the config.
func (c *CEEMSAPIAppConfig) Validate() error {
	// Validate Data config
	err := c.Server.Data.Validate()
	if err != nil {
		return err
	}

	// Validate Admin config
	err = c.Server.Admin.Validate()
	if err != nil {
		return err
	}

	return nil
}

// CEEMSAPIServerConfig contains the configuration of CEEMS API server.
type CEEMSAPIServerConfig struct {
	Data  ceems_db.DataConfig  `yaml:"data"`
	Admin ceems_db.AdminConfig `yaml:"admin"`
	Web   ceems_http.WebConfig `yaml:"web"`
}

// CEEMSServer represents the `ceems_server` cli.
type CEEMSServer struct {
	appName string
	App     kingpin.Application
}

// NewCEEMSServer creates a new CEEMSServer instance.
func NewCEEMSServer() (*CEEMSServer, error) {
	return &CEEMSServer{
		appName: base.CEEMSServerAppName,
		App:     base.CEEMSServerApp,
	}, nil
}

// Main is the entry point of the `ceems_server` command.
//
//	@title			CEEMS API
//	@version		1.0
//	@description	OpenAPI specification (OAS) for the CEEMS REST API.
//	@description
//	@description	See the Interactive Docs to try CEEMS API methods without writing code, and get
//	@description	the complete schema of resources exposed by the API.
//	@description
//	@description	If basic auth is enabled, all the endpoints require authentication.
//	@description
//	@description	All the endpoints, except `health`, `swagger`, `debug` and `demo`,
//	@description	must send a user-agent header.
//	@description
//	@description	A demo instance of CEEMS API server is provided for the users to test. This
//	@description	instance is running at `https://ceems-demo.myaddr.tools:6443` and it is the
//	@description	default server that will serve the requests originating from current OAS client.
//	@description
//	@description	Some of the valid users for this demo instance are:
//	@description	- arnold
//	@description	- betty
//	@description	- edna
//	@description	- gazoo
//	@description	- wilma
//	@description
//	@description	Every request must contain a `X-Grafana-User` header with one of the usernames
//	@description	above as the value to the header. This is how CEEMS API server recognise the user.
//	@description
//	@description	Some of the valid projects for this demo instance are:
//	@description	- bedrock
//	@description	- cornerstone
//	@description
//	@description	Demo instance have CORS enabled to allow cross-domain communication from the browser.
//	@description	All responses have a wildcard same-origin which makes them completely public and
//	@description	accessible to everyone, including any code on any site.
//	@description
//	@description	To test admin resources, users can use `admin` as `X-Grafana-User`.
//	@description
//	@description				Timestamps must be specified in milliseconds, unless otherwise specified.
//
//	@contact.name				Mahendra Paipuri
//	@contact.url				https://github.com/ceems-dev/ceems/issues
//	@contact.email				mahendra.paipuri@gmail.com
//
//	@license.name				GPL-3.0 license
//	@license.url				https://www.gnu.org/licenses/gpl-3.0.en.html
//
//	@securityDefinitions.basic	BasicAuth
//
//	@servers.url				http://localhost:9020/api/v1
//	@servers.description		Current CEEMS API server URL.
//
//	@servers.url				https://{host}:{port}/{basepath}
//	@servers.description		Current CEEMS API server URL with non default host and port.
//
//	@servers.url				https://ceems-demo.myaddr.tools:6443/api/v1
//	@servers.description		Demo CEEMS API server URL.
//
//	@externalDocs.url			https://ceems-dev.github.io/ceems/
//
//	@x-logo						{"url": "https://raw.githubusercontent.com/ceems-dev/ceems/refs/heads/main/website/static/img/logo.png", "altText": "CEEMS logo"}
func (b *CEEMSServer) Main() error {
	// CLI vars
	var (
		configFile, webConfigFile, routePrefix, corsOrigin, maxQueryPeriod, runAsUser string
		enableDebugServer, skipDeleteOldUnits, disableChecks                          bool
		dropPrivs, systemdSocket, compression, disableCapAwareness                    bool
		webListenAddresses, userHeaders, trustedIPPrefixes                            []string
		requestsLimit, maxProcs, compressionLevel                                     int
		externalURL                                                                   *url.URL
	)

	// Get default run as user
	defaultRunAsUser, err := security.GetDefaultRunAsUser()
	if err != nil {
		return err
	}

	b.App.Flag(
		"config.file",
		"Path to CEEMS API server configuration file.",
	).Envar("CEEMS_API_SERVER_CONFIG_FILE").Default("").StringVar(&configFile)
	b.App.Flag(
		"config.file.expand-env-vars",
		"Any environment variables that are referenced in config file will be expanded. To escape $ use $$ (default: false).",
	).Default("false").BoolVar(&base.ConfigFileExpandEnvVars)
	b.App.Flag(
		"web.listen-address",
		"Addresses on which to expose API server and web interface.",
	).Default(":9020").StringsVar(&webListenAddresses)
	b.App.Flag(
		"web.config.file",
		"Path to configuration file that can enable TLS or authentication. See: https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md",
	).Envar("CEEMS_API_SERVER_WEB_CONFIG_FILE").Default("").StringVar(&webConfigFile)
	b.App.Flag(
		"web.route-prefix",
		"Prefix for the internal routes of web endpoints.",
	).Default("/").StringVar(&routePrefix)
	b.App.Flag(
		"web.max-requests",
		"Maximum number of requests allowed in 1 minute period per client identified by Real IP address. "+
			"Request headers True-Client-IP, X-Real-IP and X-Forwarded-For are looked up to get the real client IP address."+
			"By default no limit is applied.",
	).Default("0").IntVar(&requestsLimit)
	b.App.Flag(
		"web.trusted-ip-prefix",
		"IP prefix of trusted IP addresses. The string can be in the form `192.168.1.0/24` or `2001:db8::/32`, the CIDR notation defined in RFC 4632 and RFC 4291. "+
			"Rate limiting will not be applied for the requests coming from these matching IP addresses.",
	).StringsVar(&trustedIPPrefixes)
	b.App.Flag(
		"web.cors.origin",
		"Regex for CORS origin. It is fully anchored. Example: 'https?://(domain1|domain2)\\.com'.",
	).Default(".*").StringVar(&corsOrigin)
	b.App.Flag(
		"web.compression",
		"Enable gzip compression for responses (default: false).",
	).Default("false").BoolVar(&compression)
	b.App.Flag(
		"web.compression.level",
		"Compression level for the responses.",
	).Default("5").IntVar(&compressionLevel)
	b.App.Flag(
		"web.debug-server",
		"Enable /debug/pprof profiling endpoints. (default: disabled).",
	).Default("false").BoolVar(&enableDebugServer)
	b.App.Flag(
		"query.max-period",
		"Maximum allowable query range. Units Supported: y, w, d, h, m, s, ms. By default no limit is applied.",
	).Default("0s").StringVar(&maxQueryPeriod)

	b.App.Flag(
		"security.run-as-user",
		"API server will be run under this user. Accepts either a username or uid. If current user is unprivileged, same user "+
			"will be used. When API server is started as root, by default user will be changed to nobody. To be able to change the user necessary "+
			"capabilities (CAP_SETUID, CAP_SETGID) must exist on the process.",
	).Default(defaultRunAsUser).StringVar(&runAsUser)

	// Hidden args that we can expose to users if found useful
	b.App.Flag(
		"web.external-url",
		"External URL at which CEEMS API server is reachable.",
	).Hidden().Default("").URLVar(&externalURL)
	b.App.Flag(
		"web.user-header-name",
		"Username will be fetched from these headers. (default: X-Grafana-User).",
	).Hidden().Default(base.GrafanaUserHeader).StringsVar(&userHeaders)

	// Testing related hidden CLI args
	b.App.Flag(
		"storage.data.skip.delete.old.units",
		"Skip deleting old compute units. Used only in testing. (default is false)",
	).Hidden().Default("false").BoolVar(&skipDeleteOldUnits)
	b.App.Flag(
		"test.disable.checks",
		"Disable sanity checks. Used only in testing. (default is false)",
	).Hidden().Default("false").BoolVar(&disableChecks)
	b.App.Flag(
		"runtime.gomaxprocs", "The target number of CPUs Go will run on (GOMAXPROCS)",
	).Envar("GOMAXPROCS").Default("1").IntVar(&maxProcs)
	b.App.Flag(
		"security.disable-cap-awareness",
		"Disable capability awareness and run as privileged process (default: false).",
	).Default("false").Hidden().BoolVar(&disableCapAwareness)
	b.App.Flag(
		"security.drop-privileges",
		"Drop privileges and run as nobody when exporter is started as root.",
	).Default("true").Hidden().BoolVar(&dropPrivs)

	// Socket activation only available on Linux
	if runtime.GOOS == "linux" {
		b.App.Flag(
			"web.systemd-socket",
			"Use systemd socket activation listeners instead of port listeners (Linux only).",
		).Default("false").BoolVar(&systemdSocket)
	}

	promslogConfig := &promslog.Config{}
	flag.AddFlags(&b.App, promslogConfig)
	b.App.Version(version.Print(b.appName))
	b.App.UsageWriter(os.Stdout)
	b.App.HelpFlag.Short('h')

	_, err = b.App.Parse(os.Args[1:])
	if err != nil {
		return fmt.Errorf("failed to parse CLI flags: %w", err)
	}

	// Parse max query period
	period, err := model.ParseDuration(maxQueryPeriod)
	if err != nil {
		return fmt.Errorf("failed to parse option for --query.max-period: %w", err)
	}

	// Compile CORS regex
	corsRegex, err := regexp.Compile("^(?s:" + corsOrigin + ")$")
	if err != nil {
		return fmt.Errorf("failed to compile option for --web.cors.origin: %w", err)
	}

	// Get absolute path for web config file if provided
	var webConfigFilePath string
	if webConfigFile != "" {
		webConfigFilePath, err = filepath.Abs(webConfigFile)
		if err != nil {
			return fmt.Errorf("failed to get absolute path of the web config file: %w", err)
		}
	}

	// Get absolute config file path global variable that will be used in resource manager
	// and updater packages
	base.ConfigFilePath, err = filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of the config file: %w", err)
	}

	// Make config from file
	config, err := common.MakeConfig[CEEMSAPIAppConfig](base.ConfigFilePath, base.ConfigFileExpandEnvVars)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	// Set directory for reading files
	config.SetDirectory(filepath.Dir(base.ConfigFilePath))
	// This is used only in tests
	config.Server.Data.SkipDeleteOldUnits = skipDeleteOldUnits

	// Return error if backup interval of less than 1 day is used
	err = config.Validate()
	if err != nil && !disableChecks {
		return err
	}

	// Setup data directories
	config, err = createDirs(config)
	if err != nil {
		return err
	}

	// Set logger here after properly configuring promlog
	logger := promslog.New(promslogConfig)

	logger.Info("Starting "+b.appName, "version", version.Info())
	logger.Info(
		"Operational information", "build_context", version.BuildContext(),
		"host_details", internal_runtime.Uname(), "fd_limits", internal_runtime.FdLimits(),
	)

	runtime.GOMAXPROCS(maxProcs)
	logger.Debug("Go MAXPROCS", "procs", runtime.GOMAXPROCS(0))

	// Make DB config.
	dbConfig := &ceems_db.Config{
		Logger:          logger,
		Data:            config.Server.Data,
		Admin:           config.Server.Admin,
		ResourceManager: resource.New,
		Updater:         updater.New,
	}

	// Make server config.
	serverConfig := &ceems_http.Config{
		Logger: logger,
		Web: ceems_http.WebConfig{
			Addresses:         webListenAddresses,
			WebSystemdSocket:  systemdSocket,
			WebConfigFile:     webConfigFilePath,
			EnableDebugServer: enableDebugServer,
			UserHeaderNames:   userHeaders,
			ExternalURL:       externalURL,
			RoutePrefix:       routePrefix,
			CORSOrigin:        corsRegex,
			EnableCompression: compression,
			CompressionLevel:  compressionLevel,
			RequestsLimit:     requestsLimit,
			TrustedIPPrefixes: trustedIPPrefixes,
			MaxQueryPeriod:    period,
			LandingConfig: &web.LandingConfig{
				Name:        b.App.Name,
				Description: b.App.Help,
				Version:     version.Info(),
				HeaderColor: "#3cc9beff",
				Links: []web.LandingLinks{
					{
						Address: "swagger/index.html",
						Text:    "Swagger API",
					},
					{
						Address: "health",
						Text:    "Health Status",
					},
				},
			},
		},
		DB: *dbConfig,
	}

	// Create server instance.
	apiServer, err := ceems_http.New(serverConfig)
	if err != nil {
		logger.Error("Failed to create CEEMS API server", "err", err)

		return err
	}

	// Create DB instance.
	collector, err := ceems_db.New(dbConfig)
	if err != nil {
		logger.Error("Failed to create CEEMS DB", "err", err)

		return err
	}

	// Make security related config
	// CEEMS API server should not need any privileges except executing SLURM sacct command.
	//
	// In future we should add SLURM API support as well which should avoid any privilege
	// requirements for CEEMS API server.
	//
	// Until then the required privileges should not be more than cap_setuid and cap_setgid.
	//
	// So we start with that assumption and during the resource manager instantitation, if
	// we are using sacct method, we keep the privilege or if the runtime config uses
	// future SLURM API support, we drop those privileges as well.
	//
	// So, we keep the privileges only as a insurance and once we confirm with resource manager
	// we decide to either keep them or drop them.
	var allCaps []cap.Value

	for _, name := range []string{"cap_setuid", "cap_setgid"} {
		value, err := cap.FromName(name)
		if err != nil {
			logger.Error("Error parsing capability %s: %w", name, err)

			continue
		}

		allCaps = append(allCaps, value)
	}

	// We should STRONGLY advise in docs that CEEMS API server should not be started as root
	// as that will end up dropping the privileges and running it as nobody user which can
	// be strange as CEEMS API server writes data to DB.
	securityCfg := &security.Config{
		RunAsUser:      runAsUser,
		Caps:           allCaps,
		ReadPaths:      []string{webConfigFilePath, base.ConfigFilePath},
		ReadWritePaths: []string{config.Server.Data.Path, config.Server.Data.BackupPath},
	}

	// If there is already a DB file, we should add it to ReadWritePaths
	dbFile := filepath.Join(config.Server.Data.Path, base.CEEMSDBName)

	_, err = os.Stat(dbFile)
	if err == nil {
		securityCfg.ReadWritePaths = append(securityCfg.ReadWritePaths, dbFile)
	}

	// Add any readPaths and readWritePaths to security config
	if len(base.AppReadPaths) > 0 {
		securityCfg.ReadPaths = append(securityCfg.ReadPaths, base.AppReadPaths...)
	}

	if len(base.AppReadWritePaths) > 0 {
		securityCfg.ReadWritePaths = append(securityCfg.ReadWritePaths, base.AppReadWritePaths...)
	}

	// Start a new manager
	securityManager, err := security.NewManager(securityCfg, logger)
	if err != nil {
		logger.Error("Failed to create a new security manager", "err", err)

		return err
	}

	// Drop all unnecessary privileges
	if dropPrivs {
		user, err := user.Current()
		if err == nil && user.Uid == "0" {
			logger.Info("CEEMS API server is running as root user. Privileges will be dropped and process will be run as unprivileged user", "become_user", runAsUser)
		}

		err = securityManager.DropPrivileges(disableCapAwareness)
		if err != nil {
			logger.Error("Failed to drop privileges", "err", err)

			return err
		}
	}

	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Declare wait group and tickers.
	var wg sync.WaitGroup

	var dbUpdateTicker, dbBackupTicker *time.Ticker

	// Initialize tickers. We will stop the ticker immediately after signal has received.
	dbUpdateTicker = time.NewTicker(time.Duration(config.Server.Data.UpdateInterval))

	wg.Go(func() {
		for {
			// This will ensure that we will run the method as soon as go routine
			// starts instead of waiting for ticker to tick.
			logger.Info("Updating CEEMS DB", "interval", config.Server.Data.UpdateInterval)

			err := collector.Collect(ctx)
			if err != nil {
				logger.Error("Failed to fetch data", "err", err)
			}

			select {
			case <-dbUpdateTicker.C:
				continue
			case <-ctx.Done():
				logger.Info("Received Interrupt. Stopping DB update")

				return
			}
		}
	})

	// Start backup go routine only backup path is provided in CLI.
	if config.Server.Data.BackupPath != "" {
		// Initialise ticker and increase waitgroup counter.
		dbBackupTicker = time.NewTicker(time.Duration(config.Server.Data.BackupInterval))

		wg.Go(func() {
			for {
				select {
				case <-dbBackupTicker.C:
					// Dont run backup as soon as go routine is spawned. In prod, it
					// can take very long depending on the size of DB and so wait until
					// first tick to run it.
					logger.Info("Backing up CEEMS DB", "interval", config.Server.Data.BackupInterval)

					err := collector.Backup(ctx)
					if err != nil {
						logger.Error("Failed to backup CEEMS DB", "err", err)
					}
				case <-ctx.Done():
					logger.Info("Received Interrupt. Stopping CEEMS DB backup")

					return
				}
			}
		})
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below.
	go func() {
		err := apiServer.Start(ctx)
		if err != nil {
			logger.Error("Failed to start CEENS API server", "err", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Stop tickers.
	dbUpdateTicker.Stop()

	if config.Server.Data.BackupPath != "" {
		dbBackupTicker.Stop()
	}

	// Wait for all DB go routines to finish.
	wg.Wait()

	// Close DB only after all DB go routines are done.
	err = collector.Stop()
	if err != nil {
		logger.Error("Failed to close CEEMS DB connection", "err", err)
	}

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	logger.Info("Shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = apiServer.Shutdown(ctx)
	if err != nil {
		logger.Error("Failed to gracefully shutdown server", "err", err)
	}

	// Restore file permissions by removing any ACLs added
	err = securityManager.DeleteACLEntries()
	if err != nil {
		logger.Error("Failed to remove ACL entries", "err", err)
	}

	logger.Info("Server exiting")
	logger.Info("See you next time!!")

	return nil
}

// createDirs makes data directories and set paths to absolute in config.
func createDirs(config *CEEMSAPIAppConfig) (*CEEMSAPIAppConfig, error) {
	var err error
	// Get absolute Data path
	config.Server.Data.Path, err = filepath.Abs(config.Server.Data.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for data.path=%s: %w", config.Server.Data.Path, err)
	}

	if config.Server.Data.BackupPath != "" {
		config.Server.Data.BackupPath, err = filepath.Abs(config.Server.Data.BackupPath)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get absolute path for data.backup_path=%s: %w",
				config.Server.Data.BackupPath,
				err,
			)
		}
	}

	// Check if config.Data.Path/config.Data.BackupPath exists and create one if it does not.
	_, err = os.Stat(config.Server.Data.Path)
	if os.IsNotExist(err) {
		err := os.MkdirAll(config.Server.Data.Path, 0o750)
		if err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	if config.Server.Data.BackupPath != "" {
		_, err := os.Stat(config.Server.Data.BackupPath)
		if os.IsNotExist(err) {
			err := os.MkdirAll(config.Server.Data.BackupPath, 0o750)
			if err != nil {
				return nil, fmt.Errorf("failed to create backup data directory: %w", err)
			}
		}
	}

	return config, nil
}
