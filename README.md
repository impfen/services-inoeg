# Kiebitz Services

This repository contains Kiebitz's backend services:

* A **storage** service that stores encrypted user & operator settings and temporary data.
* An **appointments** service that stores encrypted appointment data.

## tl;dr
**We are showing a quick version here, full details below**
make sure you installed Golang in a current version >=1.16.

we need a [Redis](#dependencies) server:  
`docker run -d -p 6379:6379 redis`

simply run
1. `make` to [install](#Installation)
1. `source .test-setup` so tests [create/setup](#basic-setup) necessary data
1. `kiebitz admin keys setup` to generate necessary [keys](#cryptographic-keys) ***will overwrite existing key files***
1. `kiebitz run all` to [run the development services](#running)

Then have a look and jump to the [APIs](#apis)

## Dependencies

The Kiebitz backend services are written in Golang. Please install a recent version, preferably >=1.16.

If you're running MacOS and run into the `readlink: illegal option -- f` error, you can resolve it by installing coreutils: `brew install coreutils` and linking it to your bin: `ln -s /usr/local/bin/greadlink /usr/local/bin/readlink`. Restart your terminal afterwards.

By default, Kiebitz uses a Redis database to store data. Please make sure a Redis server is available. You can change the connection details in the `settings/dev/001_default.yml` settings file. The metering services (for statistics) also uses a Redis database by default and can be configured just like the main database. In addition, to generate TLS certificates (which is not always necessary) Kiebitz relies on the `openssl` CLI.

## Installation

To build and install all services into your `GOPATH`, simply run

```bash
make
```

## Basic Setup

Kiebitz needs cryptographic keys and some helper data to function correctly. All of these can be generated using the `kiebitz` command.

First things first: Kiebitz looks for settings in the path given by the `KIEBITZ_SETTINGS` environment variable. For development, settings are in the `settings/dev` subdirectory of the repository. To set up our development environment, we simply run

```bash
source .dev-setup
```

For the commands below we'll assume that `KIEBITZ_SETTINGS` points to the `settings/dev` directory.

### Cryptographic Keys

Now, we need to generate various cryptographic keys for signing and encryption. To do this, we simply run

```bash
kiebitz admin keys setup
```

**Warning:** Running this command will overwrite existing key files, potentially rendering all your development data useless, so be careful.

This will generate two files in the Kiebitz settings directory, `002_admin.json` and `003_appt.json`. The former is only for administration purposes and should remain locked away. The latter is for use with the appointments server.

Optionally, you can encrypt the `002_admin.json` file with a passphrase. The
passphrase must be present in the `KIEBITZ_PASSPHRASE` environment variable, and
has to be pretsent for the generation of the keys as well as **every time** the
server starts. You can generate an encrypted key file with the `-e` switch like
this:

```bash
kiebitz admin keys setup -e
```

Now we can then generate mediator keys. To do this, we simply run

```bash
kiebitz admin keys mediator > data/secret-mediator-keys.json
```

This will create a JSON data structure with all necessary keys for the mediator. Please be aware that the `provider` keys are just copied from the `002_admin.json` file generated before. These keys enable mediators to decrypt provider data.

For the next steps of the setup process we'll need a running backend, so let's start it via

```bash
kiebitz --level debug run all
```

That should start the appointments and storage services. Now we can tell the backend about the newly created mediator keys, which we do via

```bash
kiebitz admin mediators upload data/secret-mediator-keys.json
```

This will sign the public signing and encryption keys of the mediator with the root key and put the signed key material on the backend for publication. That's it! Now we should be able to go to the `/mediator` URL in the frontend, load our mediator key file and verify providers. Providers should be able to sign up, upload their data for verification and get tokens. Users should also be able to sign up and receive invitations.

### ZIP Code Data

ZIP code data helps Kiebitz to estimate distances between zip code areas. There are two files `data/distances.json` and `data/distances-areas.json` that need to be uploaded. We can do this via

```bash
# upload distances for full ZIP codes (used when matching tokens to providers)
kiebitz admin distances upload data/distances.json
# upload distances for ZIP codes areas (used when filtering appointments for users)
kiebitz admin distances upload data/distances-areas.json
```

To generate the distances, we can use the `make_distances.py` and `make_area_distances.py` scripts from the `.scripts` folder (normally this is not necessary though):

```bash
python3 .scripts/make_distances.py
python3 .scripts/make_area_distances.py
```

**Note:** We can test the system without the ZIP code data, but tokens will then only be distributed to matching zip codes.

## Signup Codes

We can also upload user & provider codes if we want to restrict who can register on the platform (this requires setting `appointments.user_codes_enabled: true` and `appointments.provider_codes.enabled: true`, respectively):

```bash
# upload user codes from a file
kiebitz admin codes upload data/secret-user-codes.json
# upload provider codes from a file
kiebitz admin codes upload data/secret-provider-codes.json
```

To generate codes, we simply run

```bash
# generate 10.000 user codes
kiebitz admin codes generate --actor user -n 10000 > data/secret-user-codes.json
# generate 10.000 provider codes
kiebitz admin codes generate --actor provider -n 10000 > data/secret-user-codes.json
```

Codes are just random 16 byte values, and the `actor` parameter just tells the backend for which actor the codes should be used.

### TLS Certificates

Finally, if we want to run Kiebitz using a self-signed TLS certificate, we simply run

```bash
make certs
```

to generate these certificates, and then enable them by commenting out the `tls` section in the settings.

## Running

Start redis:
```bash
docker run -d -p 6379:6379 redis
```

To run the development services we can then

```bash
# run the appointments service
kiebitz run appointments
# ...or run the storage service
kiebitz run storage
# ...or run all services
kiebitz run all
```

## APIs

The Kiebitz services can be exposed as a JSON-RPC or REST service (or both). For example, if both API types are enabled, the `getAppointmentsByZipCode` endpoint with parameters `zipCode=10707` and `radius=20` can be reached via both of the following queries:

```bash
# REST endpoint
curl http://localhost:8888/appointments/zipCode/10707/20
# JSON-RPC endpoint
curl -X POST --header "Content-Type: application/json" http://localhost:8888/jsonrpc --data '{"jsonrpc": "2.0", "method": "getAppointmentsByZipCode", "params": {"zipCode": "10707", "radius": "20"}}'
```

In general, the REST API is better for caching as it exposes cacheable endpoints via GET requests, while the JSON-RPC API provides a simpler and more natural interface.

## Testing

Here's how you can send a request to the storage server via `curl` (this assumes you have `jq` installed for parsing of the JSON result):

```bash
curl --cacert settings/dev/certs/root.crt --resolve storage-1:9999:127.0.0.1 https://storage-1:9999/jsonrpc --header "Content-Type: application/json; charset=utf-8" --data '{"method": "getSettings", "id": "2", "params": {"key": "az4df7vjunsd6ad"}, "jsonrpc": "2.0"}' 2>/dev/null | jq
```

To run all Go tests and benchmarks, simply

```bash
# run normal tests
make test
# run race-condition tests
make test-races
# run benchmarks
make bench
```

### Load Testing

**Careful, the following commands will create massive amounts of fake data via the API client, never run this against the production system, except for initial load testing!**

You can simulate providers, appointments and bookings in the backend using the `testing benchmark` command. The following command will create 1.000 providers with 1.000 appointments with 20 slots each, hence a total of 20.000.000 appointments:

```bash
kiebitz testing benchmark --providers 1000 --appointments 1000 --slots 20
```

This command will use the given settings and connect to a running API server, hence to run it you need to also run the Kiebitz API. Probably you want to use the `test` settings for this

## Development

To auto-generate copyright headers for Golang files, simply run

```bash
make copyright
```

## Security

Did you find a security issue you'd like to report? Please contact us at [security@kiebitz.eu](mailto:security@kiebitz.eu). We appreciate if you follow [responsible disclosure practices](https://en.wikipedia.org/wiki/Responsible_disclosure).
**Please do not create a public Github issue for exploitable vulnerabilities that you've identified**. We do not have a formal bug bounty program in place but will try to reward researchers for significant security findings. We also pledge to never take any kind of legal action or retaliate against researchers that disclose security issues in good faith. You may of course disclose issues anonymously as well.
