# KMFDDM Operations Guide

This is a brief technical overview of the KMFDDM server and related tools.

Refer to the [project README](../README.md) for a conceptual introduction to KMFDDM and the [Quickstart Guide](quickstart.md) can be used for getting a basic environment up and running. This document is intended for more operational details.

## kmfddm

### Command line flags

Command line flags can be specified using command line arguments or environment variables (in KMFDDM versions later than v0.8.1). Flags take precedence over environment variables, which take precedence over default values. Environment variables are denoted in square brackets below (e.g., [HELLO]), and default values are shown in parentheses (e.g., (default "world")). If an environment variable is currently set then the help output will add "is set" as an indicator.

#### -h, -help

Built-in flag that prints all other available flags, environment variables, and defaults.

#### -api string

* API key for API endpoints [KMFDDM_API]

Required. API authentication in KMFDDM is simply HTTP Basic authentication using "kmfddm" as the username and the API key (from this flag) as the password.

#### -cors-origin string

* CORS Origin; for browser-based API access [KMFDDM_CORS_ORIGIN]

Sets CORS origin and related HTTP headers on requests.

#### -debug

* log debug messages [KMFDDM_DEBUG]

Enable additional debug logging.

#### -dump-status string

* file name to dump status reports to ("-" for stdout) [KMFDDM_DUMP_STATUS]

KMFDDM supports dumping the JSON Declarative Device Management status report to a file. Specify a dash (`-`) to dump to stdout.

#### -enqueue string

* URL of MDM server enqueue endpoint [KMFDDM_ENQUEUE]

URL of the MDM server for enqueuing commands. The enrollmnet ID is added onto this URL as a path element (or multiple, if the MDM server supports it).

#### -enqueue-key string

* MDM server enqueue API key [KMFDDM_ENQUEUE_KEY]

The API key (HTTP Basic authentication password) for the MDM server enqueue endpoint. The HTTP Basic username depends on the MDM mode. By default it is "nanomdm" but if the `-micromdm` (see below) flag is enabled then it is "micromdm".

#### -listen string

* HTTP listen address [KMFDDM_LISTEN] (default ":9002")

Specifies the listen address (interface and port number) for the server to listen on.

#### -micromdm

* Use MicroMDM command API calling conventions [KMFDDM_MICROMDM]

Submit commands for enqueueing in a style that is compatible with MicroMDM (instead of NanoMDM). Specifically this flag limits sending commands to one enrollment ID at a time, uses a POST request, and changes the HTTP Basic username.

### -shard

* enable shard management properties declaration [KMFDDM_SHARD]

Enable an always-on [management properties declaration](https://developer.apple.com/documentation/devicemanagement/managementproperties) for every enrollment. It contains a `shard` payload key which is a dynamically computed integer between 0 and 100, inclusive, based on the enrollment ID. This `shard` key can then be used in activation declaration predicates. For example `(@property(shard) <= 75)`. The identifier of this dynamic declaration is `com.github.jessepeterson.kmfddm.storage.shard.v1`; the Server Token includes the shard number. It is "static" in that it should not change for any given enrollment.

### -storage, -storage-dsn, & -storage-options

* -storage string
  * storage backend [KMFDDM_STORAGE] (default "filekv")
* -storage-dsn string
  * storage data source name [KMFDDM_STORAGE_DSN]
* -storage-options string
  * storage backend options [KMFDDM_STORAGE_OPTIONS]

The `-storage`, `-storage-dsn`, & `-storage-options` flags together configure the storage backend. `-storage` specifies the name of the backend while `-storage-dsn` specifies the backend data source name (e.g. the connection string). The optional `-storage-options` flag specifies options for the backend (if it supports them). If no storage flags are supplied then it is as if you specified `-storage filekv -storage-dsn dbkv` meaning we use the `filekv` storage backend with `dbkv` as its DSN.

#### filekv storage backend

* `-storage filekv`

Configures the `filekv` storage backend. This manages storing data within plain filesystem files and directories using a key-value storage system. It has zero dependencies and should run out of the box. The `-storage-dsn` flag specifies the filesystem directory for the database otherwise `dbkv` is used. The `filekv` backend has no options.

*Example* `-storage filekv -storage-dsn /path/to/my/db`

#### mysql storage backend

* `-storage mysql`

Configures the MySQL storage backend. The `-storage-dsn` flag should be in the [format the SQL driver expects](https://github.com/go-sql-driver/mysql#dsn-data-source-name). Be sure to create your tables with the [schema.sql](../storage/mysql/schema.sql) file that corresponds to your KMFDDM version. Also make sure you apply any schema changes for each updated version (i.e. execute the numbered schema change files). MySQL 8.0.19 or later is required.

*Example:* `-storage mysql -storage-dsn kmfddm:kmfddm/mymdmdb`

Options are specified as a comma-separated list of "key=value" pairs. The mysql backend supports these options:

* `delete_errors=N`
  * This option sets the maximum number of errors to keep in the database per enrollment ID. A default of zero means to store unlimited errors in the database for each enrollment.
* `delete_status_reports=N`
  * This option sets the maximum number of errors to keep in the database per enrollment ID. A default of zero means to store unlimited errors in the database for each enrollment.

*Example:* `-storage mysql -storage-dsn kmfddm:kmfddm/mymdmdb -storage-options delete_errors=20,delete_status_reports=5`

#### in-memory storage backend

* `-storage inmem`

Configure the `inmem` in-memory storage backend. This manages DDM data entirely in *volatile* memeory. There are no options and the DSN is ignored.

> [!CAUTION]
> All data is lost when the server process exits when using the in-memory storage backend.

*Example:* `-storage inmem`

#### file storage backend

* `-storage file`

> [!WARNING]
> The `file` storage backend is deprecated in KMFDDM **versions after v0.7** and will be removed in a future release.

Configures the `file` storage backend. This manages storage data within plain filesystem files and directories.  It has zero dependencies but is disabled out of the box. The `-storage-dsn` flag specifies the filesystem directory for the database.

Options are specified as a comma-separated list of "key=value" pairs. Supported options:

* `enable_deprecated=1`
  * This option enables the file backend. Without this flag the `file` backend is disabled.

*Example:* `-storage file -storage-dsn /path/to/my/db -storage-options enable_deprecated=1`

#### -version

* print version and exit

Print version and exit.

## Tools and scripts

The KMFDDM project includes tools and scripts that use the HTTP API for configuration. Most of these are basically just shell scripts that utilize `curl` and `jq` to assist in managing the KMFDDM server.
