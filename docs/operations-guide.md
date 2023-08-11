# KMFDDM Operations Guide

This is a brief technical overview of the KMFDDM server and related tools.

Refer to the [project README](../README.md) for a conceptual introduction to KMFDDM and the [Quickstart Guide](quickstart.md) can be used for getting a basic environment up and running. This document is intended for more operational details.

## kmfddm

### Switches

#### -version

* print version

Print version and exit.

#### -api string

 * API key for API endpoints

Required. API authentication in NanoDEP is simply HTTP Basic authentication using "kmfddm" as the username and the API key (from this switch) as the password.

#### -cors-origin string

 * CORS Origin; for browser-based API access

Sets CORS origin and related HTTP headers on requests.

#### -debug

 * log debug messages

Enable additional debug logging.

#### -dump-status string

 * file name to dump status reports to ("-" for stdout)

KMFDDM supports dumping the JSON Declarative Device Management status report to a file. Specify a dash (`-`) to dump to stdout.

#### -enqueue string

 * URL of MDM server enqueue endpoint

URL of the MDM server for enqueuing commands. The enrollmnet ID is added onto this URL as a path element (or multiple, if the MDM server supports it).

#### -enqueue-key string

 * MDM server enqueue API key

The API key (HTTP Basic authentication password) for the MDM server enqueue endpoint. The HTTP Basic username depends on the MDM mode. By default it is "nanomdm" but if the `-micromdm` (see below) flag is enabled then it is "micromdm".

#### -listen string

 * HTTP listen address (default ":9002")

Specifies the listen address (interface and port number) for the server to listen on.

#### -micromdm

 * Use MicroMDM command API calling conventions

Submit commands for enqueueing in a style that is compatible with MicroMDM (instead of NanoMDM). Specifically this flag limits sending commands to one enrollment ID at a time, uses a POST request, and changes the HTTP Basic username.

### -storage, -storage-dsn, & -storage-options

The `-storage`, `-storage-dsn`, & `-storage-options` flags together configure the storage backend. `-storage` specifies the name of the backend while `-storage-dsn` specifies the backend data source name (e.g. the connection string). The optional `-storage-options` flag specifies options for the backend (if it supports them). If no storage flags are supplied then it is as if you specified `-storage file -storage-dsn db` meaning we use the `file` storage backend with `db` as its DSN.

#### file storage backend

* `-storage file`

Configures the `file` storage backend. This manages storage data within plain filesystem files and directories. It has zero dependencies and should run out of the box. The `-storage-dsn` flag specifies the filesystem directory for the database. The `file` backend has no storage options.

*Example:* `-storage file -storage-dsn /path/to/my/db`

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