
[![Build and Test](https://github.com/wrouesnel/multihttp/actions/workflows/integration.yml/badge.svg)](https://github.com/wrouesnel/multihttp/actions/workflows/integration.yml)
[![Coverage Status](https://coveralls.io/repos/github/wrouesnel/multihttp/badge.svg?branch=main)](https://coveralls.io/github/wrouesnel/multihttp?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/wrouesnel/multihttp)](https://goreportcard.com/report/github.com/wrouesnel/multihttp)

# MultiHTTP - Easily start multiple HTTP listeners

Usage
```
go get github.com/wrouesnel/multihttp@v1
```

Simple library to allow easily starting multiple HTTP/HTTPS listener services.

It supports both HTTP and HTTPS servers, and allows specifying different
certificate packages for each HTTPS listener.

Servers are started by specifying addresses with URL-like schemas:

* `unix:///var/run/server.socket` : open a Unix socket file on /var/run/server
* `tcp://0.0.0.0:80` : listen on tcp port 80.
* `tcps://0.0.0.0:443?tlscert=/path/to/file/in/pem/format.crt&tlskey=/path/to/file/in/pem/format.pem`
* `unixs:///var/run/server.socket?tlscert=/path/to/file/in/pem/format.crt&tlskey=/path/to/file/in/pem/format.pem`

With TLS client authentication:

* `tcps://0.0.0.0:443?tlscert=/path/to/file/in/pem/format.crt&tlskey=/path/to/file/in/pem/format.pem&tlsclientca=/path/to/cert`

## Configuration

### Supported Schemes

* `tcp` - unencrypted TCP listener
* `tcps` - TLS TCP listener
* `unix` - unencrypted unix listener
* `unixs` - TLS UNIX socket listener (useful when doing SNI redirect with a server like _nginx_ or _haproxy_)

## URI configuration arguments

| Name                    | Description                                                                                                                                                                                                                                                                                        | Default |
|-------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----|
| `tlscert`               | Path to the TLS certificate for a TLS listener type                                                                                                                                                                                                                                                |  |   
| `tlskey`                | Path to the TLS key for a TLS listener type                                                                                                                                                                                                                                                        |     |
| `tlsclientca`           | Path to the TLS client CA certificate for global authentication. <br/><br/>May contain multiple certificates - all will be accepted.<br/><br/>This is watched file on disk and will be reloaded when changed.<br/><br/>While the file must exist to not be a configuration error, it may be empty. |     |
| `tlsclientcadir`        | Path to a directory of client CA certificates for global authentication.<br/><br/>The directory on disk will be watched and reloaded when changed.<br/>Client CA dirs are inherently optional - an empty dir however will not allow any clients to connect.                                        |         |
| `tlsclientcertrequest`  | Boolean which forces a TLS client to request an optional client CA, making it available on the listener.                                                                                                                                                                                           |         |
