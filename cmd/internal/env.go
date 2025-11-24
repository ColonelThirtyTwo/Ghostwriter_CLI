package internal

// Functions for managing the environment variables that control the
// configuration of the Ghostwriter containers.

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

// Configuration is a custom type for storing configuration values as Key:Val pairs.
type Configuration struct {
	Key string
	Val string
}

// Configurations is a custom type for storing `Configuration` values
type Configurations []Configuration

// Len returns the length of a Configurations struct
func (c Configurations) Len() int {
	return len(c)
}

// Less determines if one Configuration is less than another Configuration
func (c Configurations) Less(i, j int) bool {
	return c[i].Key < c[j].Key
}

// Swap exchanges the position of two Configuration values in a Configurations struct
func (c Configurations) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Initialize the environment variables.
var GhostEnv = viper.New()

// Set sane defaults for a basic Ghostwriter deployment.
// Defaults are geared towards a development environment.
func setGhostwriterConfigDefaultValues() {
	// Project configuration
	GhostEnv.SetDefault("use_docker", "yes")
	GhostEnv.SetDefault("ipythondir", "/app/.ipython")

	// Django configuration
	GhostEnv.SetDefault("django_mfa_always_reveal_backup_tokens", false)
	GhostEnv.SetDefault("django_account_allow_registration", false)
	GhostEnv.SetDefault("django_account_reauthentication_timeout", 32400)
	GhostEnv.SetDefault("django_account_email_verification", "none")
	GhostEnv.SetDefault("django_admin_url", "admin/")
	GhostEnv.SetDefault("django_allowed_hosts", "localhost 127.0.0.1 django nginx host.docker.internal ghostwriter.local")
	GhostEnv.SetDefault("django_compress_enabled", true)
	GhostEnv.SetDefault("django_csrf_cookie_secure", false)
	GhostEnv.SetDefault("django_csrf_trusted_origins", "")
	GhostEnv.SetDefault("django_date_format", "d M Y")
	GhostEnv.SetDefault("django_host", "django")
	GhostEnv.SetDefault("django_jwt_secret_key", GenerateRandomPassword(32, false))
	GhostEnv.SetDefault("django_mailgun_api_key", "")
	GhostEnv.SetDefault("django_mailgun_domain", "")
	GhostEnv.SetDefault("django_port", "8000")
	GhostEnv.SetDefault("django_qcluster_name", "soar")
	GhostEnv.SetDefault("django_secret_key", GenerateRandomPassword(32, false))
	GhostEnv.SetDefault("django_secure_ssl_redirect", false)
	GhostEnv.SetDefault("django_session_cookie_age", 32400)
	GhostEnv.SetDefault("django_session_cookie_secure", false)
	GhostEnv.SetDefault("django_session_expire_at_browser_close", false)
	GhostEnv.SetDefault("django_session_save_every_request", true)
	GhostEnv.SetDefault("django_settings_module", "config.settings.local")
	GhostEnv.SetDefault("django_social_account_allow_registration", false)
	GhostEnv.SetDefault("django_social_account_domain_allowlist", "")
	GhostEnv.SetDefault("django_social_account_login_on_get", false)
	GhostEnv.SetDefault("django_superuser_email", "admin@ghostwriter.local")
	GhostEnv.SetDefault("django_superuser_password", GenerateRandomPassword(32, true))
	GhostEnv.SetDefault("django_superuser_username", "admin")
	GhostEnv.SetDefault("django_web_concurrency", 4)

	// PostgreSQL configuration
	GhostEnv.SetDefault("postgres_host", "postgres")
	GhostEnv.SetDefault("postgres_port", 5432)
	GhostEnv.SetDefault("postgres_db", "ghostwriter")
	GhostEnv.SetDefault("postgres_user", "postgres")
	GhostEnv.SetDefault("postgres_password", GenerateRandomPassword(32, true))
	GhostEnv.SetDefault("POSTGRES_CONN_MAX_AGE", 0)

	// Redis configuration
	GhostEnv.SetDefault("redis_host", "redis")
	GhostEnv.SetDefault("redis_port", 6379)

	// Nginx configuration
	GhostEnv.SetDefault("nginx_host", "nginx")
	GhostEnv.SetDefault("nginx_port", 443)

	// Hasura configuration
	GhostEnv.SetDefault("hasura_graphql_action_secret", GenerateRandomPassword(32, true))
	GhostEnv.SetDefault("hasura_graphql_admin_secret", GenerateRandomPassword(32, true))
	GhostEnv.SetDefault("hasura_graphql_dev_mode", true)
	GhostEnv.SetDefault("hasura_graphql_enable_console", false)
	GhostEnv.SetDefault("hasura_graphql_enabled_log_types", "startup, http-log, webhook-log, websocket-log, query-log")
	GhostEnv.SetDefault("hasura_graphql_enable_telemetry", false)
	GhostEnv.SetDefault("hasura_graphql_server_host", "graphql_engine")
	GhostEnv.SetDefault("hasura_graphql_server_hostname", "graphql_engine")
	GhostEnv.SetDefault("hasura_graphql_insecure_skip_tls_verify", true)
	GhostEnv.SetDefault("hasura_graphql_log_level", "warn")
	GhostEnv.SetDefault("hasura_graphql_metadata_dir", "/metadata")
	GhostEnv.SetDefault("hasura_graphql_migrations_dir", "/migrations")
	GhostEnv.SetDefault("hasura_graphql_server_port", 8080)

	// Docker & Django health check configuration
	GhostEnv.SetDefault("healthcheck_disk_usage_max", 90)
	GhostEnv.SetDefault("healthcheck_interval", "300s")
	GhostEnv.SetDefault("healthcheck_mem_min", 100)
	GhostEnv.SetDefault("healthcheck_retries", 3)
	GhostEnv.SetDefault("healthcheck_start", "60s")
	GhostEnv.SetDefault("healthcheck_timeout", "30s")

	// Set some helpful aliases for common settings
	GhostEnv.RegisterAlias("date_format", "django_date_format")
	GhostEnv.RegisterAlias("admin_password", "django_superuser_password")
	GhostEnv.RegisterAlias("hasura_password", "hasura_graphql_admin_secret")
}

// WriteGhostwriterEnvironmentVariables writes the environment variables to the ".env" file.
func WriteGhostwriterEnvironmentVariables() {
	c := GhostEnv.AllSettings()
	// To make it easier to read and look at, get all the keys, sort them, and display variables in order
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	f, err := os.Create(filepath.Join(GetCwdFromExe(), ".env"))
	if err != nil {
		log.Fatalf("Error writing out environment!\n%v", err)
	}
	defer f.Close()
	for _, key := range keys {
		if len(GhostEnv.GetString(key)) == 0 {
			_, err = f.WriteString(fmt.Sprintf("%s=\n", strings.ToUpper(key)))
		} else {
			_, err = f.WriteString(fmt.Sprintf("%s='%s'\n", strings.ToUpper(key), GhostEnv.GetString(key)))
		}

		if err != nil {
			log.Fatalf("Failed to write out environment!\n%v", err)
		}
	}
}

// ParseGhostwriterEnvironmentVariables attempts to find and open an existing .env file or create a new one.
// If an .env file is found, load it into the Viper configuration.
// If an .env file is not found, create a new one with default values.
// Then write the final file with "WriteGhostwriterEnvironmentVariables()".
func ParseGhostwriterEnvironmentVariables() {
	setGhostwriterConfigDefaultValues()
	GhostEnv.SetConfigName(".env")
	GhostEnv.SetConfigType("env")
	GhostEnv.AddConfigPath(GetCwdFromExe())
	GhostEnv.AutomaticEnv()
	// Check if expected env file exists
	if !FileExists(filepath.Join(GetCwdFromExe(), ".env")) {
		_, err := os.Create(filepath.Join(GetCwdFromExe(), ".env"))
		if err != nil {
			log.Fatalf("The .env doesn't exist and couldn't be created")
		}
	}
	// Try reading the env file
	if err := GhostEnv.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("Error while reading in .env file: %s", err)
		} else {
			log.Fatalf("Error while parsing .env file: %s", err)
		}
	}
	WriteGhostwriterEnvironmentVariables()
}

// SetProductionMode updates the environment variables to switch to production mode.
func SetProductionMode() {
	GhostEnv.Set("hasura_graphql_dev_mode", false)
	GhostEnv.Set("django_secure_ssl_redirect", true)
	GhostEnv.Set("django_settings_module", "config.settings.production")
	GhostEnv.Set("django_csrf_cookie_secure", true)
	GhostEnv.Set("django_session_cookie_secure", true)
	WriteGhostwriterEnvironmentVariables()
}

// SetDevMode updates the environment variables to switch to development mode.
func SetDevMode() {
	GhostEnv.Set("hasura_graphql_dev_mode", true)
	GhostEnv.Set("django_secure_ssl_redirect", false)
	GhostEnv.Set("django_settings_module", "config.settings.local")
	GhostEnv.Set("django_csrf_cookie_secure", false)
	GhostEnv.Set("django_session_cookie_secure", false)
	WriteGhostwriterEnvironmentVariables()
}

// Convert the environment variable ("env") to a slice of strings.
func splitVariable(env string) []string {
	return strings.Split(GhostEnv.GetString(env), " ")
}

// Remove one or more matches for "item" from a "slice" of strings.
func removeItem(slice []string, item string) []string {
	counter := 0
	// We loop through the entire list in case an exact match appears more than once
	for i, v := range slice {
		if strings.TrimSpace(v) != item {
			slice[counter] = slice[i]
			counter++
		}
	}
	slice = slice[:counter]
	return slice
}

// Append a "host" to the given environment variable ("env").
func appendHost(env string, host string) {
	s := splitVariable(env)
	// Append the new host only if it's not already in the list
	if !(Contains(s, host)) {
		s = append(s, host)
		GhostEnv.Set(env, strings.TrimSpace(strings.Join(s, " ")))
	} else {
		log.Printf("Host %s is already in the list", host)
	}
}

// Remove a "host" from the given environment variable ("env").
func removeHost(env string, host string) {
	s := splitVariable(env)
	s = removeItem(s, host)
	GhostEnv.Set(env, strings.TrimSpace(strings.Join(s, " ")))
}

// GetConfigAll retrieves all values from the .env configuration file.
func GetConfigAll() Configurations {
	c := GhostEnv.AllSettings()
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var values Configurations
	for _, key := range keys {
		val := GhostEnv.GetString(key)
		values = append(values, Configuration{strings.ToUpper(key), val})
	}

	sort.Sort(values)

	return values
}

// GetConfig retrieves the specified values from the .env file.
func GetConfig(args []string) Configurations {
	var values Configurations
	for i := 0; i < len(args[0:]); i++ {
		setting := strings.ToLower(args[i])
		val := GhostEnv.GetString(setting)
		if val == "" {
			log.Fatalf("Config variable `%s` not found", setting)
		} else {
			values = append(values, Configuration{setting, val})
		}
	}

	sort.Sort(values)

	return values
}

// SetConfig sets the value of the specified key in the .env file.
func SetConfig(key string, value string) {
	if strings.ToLower(value) == "true" {
		GhostEnv.Set(key, true)
	} else if strings.ToLower(value) == "false" {
		GhostEnv.Set(key, false)
	} else {
		GhostEnv.Set(key, value)
	}
	WriteGhostwriterEnvironmentVariables()
}

// AllowHost appends a host to the allowed hosts list in the .env file.
func AllowHost(host string) {
	appendHost("django_allowed_hosts", host)
	WriteGhostwriterEnvironmentVariables()
}

// DisallowHost removes a host to the allowed hosts list in the .env file.
func DisallowHost(host string) {
	removeHost("django_allowed_hosts", host)
	WriteGhostwriterEnvironmentVariables()
}

// TrustOrigin appends an origin to the trusted origins list in the .env file.
func TrustOrigin(host string) {
	appendHost("django_csrf_trusted_origins", host)
	WriteGhostwriterEnvironmentVariables()
}

// DistrustOrigin removes an origin to the trusted origins list in the .env file.
func DistrustOrigin(host string) {
	removeHost("django_csrf_trusted_origins", host)
	WriteGhostwriterEnvironmentVariables()
}
