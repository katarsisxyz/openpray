# OpenPray

[![CI](https://github.com/katarsisxyz/openpray/actions/workflows/ci.yml/badge.svg)](https://github.com/katarsisxyz/openpray/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/katarsisxyz/openpray/prayer.svg)](https://pkg.go.dev/github.com/katarsisxyz/openpray/prayer)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Islamic prayer times, **calculated astronomically**. OpenPray does not depend on a third-party API and uses only the Go standard library.

The built-in methods are calibrated against official Indonesian schedules:

| Method | Reference | Exact match | Remaining difference |
|---|---|---|---|
| `kemenag` | Ministry of Religious Affairs (Kemenag RI): 5 cities, 4 months, 4,920 times | **92.8%** | ≤ 1 minute |
| `muhammadiyah` | Muhammadiyah KHGT: 5 cities, September–October 2026, 1,830 times | **97.3%** | ≤ 1 minute |

For Kemenag, most of the remaining difference comes from the reference point Kemenag uses for each city. Kemenag publishes schedules per city or regency but does not publish that point. OpenPray calculates for the exact coordinates you give it, or for the capital of one of Indonesia's 514 regencies and cities (see [Cities](#cities)).

The Muhammadiyah parameters were fitted on September data only. On October, which was not used for fitting, they matched 98.1% of times.

There are three ways to use OpenPray:

| Option | Best for |
|---|---|
| [Go library](#1-go-library) | Go services that calculate prayer times in-process |
| [HTTP API](#2-http-api) | Services in any language (self-hosted with Docker) |
| [CLI](#3-cli) | Quick checks from a terminal |

## 1. Go library

```sh
go get github.com/katarsisxyz/openpray/prayer
```

```go
loc, _ := time.LoadLocation("Asia/Jakarta")
date := time.Date(2026, 9, 30, 0, 0, 0, 0, loc)

t, err := prayer.Calculate(date, prayer.Coordinates{Lat: -6.1754, Lng: 106.8272}, prayer.Kemenag)
if err != nil {
	// prayer.ErrUndefined at high latitudes
}
fmt.Println(t.Fajr.Format("15:04")) // 04:24
```

Results are `time.Time` values in `date.Location()`.

A method is a plain struct, so you can build your own. For example, to follow a local mosque that runs Maghrib 2 minutes later:

```go
m := prayer.Kemenag
m.AsrFactor = 2                // Hanafi Asr
m.Adjust.Maghrib = 2           // per-prayer minute adjustments
```

## 2. HTTP API

```sh
docker build -t openpray .
docker run -p 8080:8080 openpray
# or without Docker:
go run ./cmd/server
```

The image is about 9 MB. It is built on distroless, runs as a non-root user, and bundles its own timezone data.

| Endpoint | Description |
|---|---|
| `GET /v1/times?(city= \| lat=&lng=&tz=)[&date=][&method=]` | Times for one day (defaults to today) |
| `GET /v1/times/month?(city= \| lat=&lng=&tz=)&year=&month=[&method=]` | Times for a whole month |
| `GET /v1/cities?[q=][&limit=]` | Search Indonesian regencies and cities |
| `GET /v1/cities/{id}` | One regency or city |
| `GET /v1/methods` | Available calculation methods and their parameters |
| `GET /openapi.yaml` | OpenAPI 3.1 spec, for Swagger UI or SDK generation |
| `GET /healthz` | Liveness check |

```sh
curl "localhost:8080/v1/times?lat=-6.1754&lng=106.8272&tz=Asia/Jakarta&date=2026-09-30"
```

```json
{
  "location": { "lat": -6.1754, "lng": 106.8272, "elevation": 0, "timezone": "Asia/Jakarta" },
  "method": { "id": "kemenag", "name": "Kemenag RI" },
  "date": "2026-09-30",
  "times": {
    "imsak": "04:14", "fajr": "04:24", "sunrise": "05:35", "dhuha": "06:02",
    "dhuhr": "11:46", "asr": "14:52", "maghrib": "17:50", "isha": "18:59"
  }
}
```

A place is given in one of two ways:
- **`city`**: the Kemendagri code of an Indonesian regency or city, such as `35.78` for Surabaya. The coordinates and time zone come from the city.
- **`lat` + `lng` + `tz`** (+ optional `elev` in meters): any location. `tz` must be an IANA name, such as `Asia/Jakarta`, `Asia/Makassar` or `Asia/Jayapura`. OpenPray does not guess the time zone from longitude, because a guess can be an hour off near the borders between Indonesian time zones.

Invalid input returns `400 {"error": "..."}`, and an unknown city returns `404`.

### Cities

`/v1/cities` covers all 514 regencies (*kabupaten*) and cities (*kota*) of Indonesia, identified by their Kemendagri code. The search matches every word of `q` against the name, the capital and the province, and puts exact name matches first:

```sh
curl "localhost:8080/v1/cities?q=bogor"
```

```json
{
  "cities": [
    { "id": "32.01", "name": "Kabupaten Bogor", "capital": "Cibinong", "province": "Jawa Barat",
      "lat": -6.479479, "lng": 106.824717, "elevation": 134, "timezone": "Asia/Jakarta" },
    { "id": "32.71", "name": "Kota Bogor", "capital": "Bogor", "province": "Jawa Barat",
      "lat": -6.595057, "lng": 106.794112, "elevation": 260, "timezone": "Asia/Jakarta" }
  ]
}
```

```sh
curl "localhost:8080/v1/times?city=32.71"
```

Prayer times for a city are calculated at the location of its capital. The city data comes from [cahyadsn/wilayah](https://github.com/cahyadsn/wilayah) (Kepmendagri 2025, MIT license). `go generate ./internal/repository` regenerates it from a pinned version of that dataset.

The server reads one setting from the environment: `PORT` (default `8080`).

To generate a client for another language from the spec, you can use [openapi-generator](https://openapi-generator.tech):

```sh
openapi-generator-cli generate -i api/openapi.yaml -g typescript-fetch -o client-ts
```

## 3. CLI

```sh
go install github.com/katarsisxyz/openpray/cmd/openpray@latest
openpray -lat -6.1754 -lng 106.8272 -tz Asia/Jakarta -method muhammadiyah
```

## Calculation methods

| ID | Fajr | Isha | Sunrise/sunset horizon | Ihtiyat | Adjustments | Reference |
|---|---|---|---|---|---|---|
| `kemenag` (default) | 20° | 18° | 1.0° | 2 min | Dhuhr +1 | Kemenag RI |
| `muhammadiyah` | 18° | 18° | 1.08° | 1 min | Sunrise −1 | Muhammadiyah KHGT |
| `mwl` | 18° | 17° | 0.833° + elevation | – | – | Muslim World League |

Terms used in the table and in the API:
- **Ihtiyat** is a precautionary margin. Sunrise moves earlier by this margin, and every other time moves later by it. Times are then rounded the way Kemenag does it.
- **Adjustments** are extra per-prayer minute corrections, applied after ihtiyat.
- **Imsak** is Fajr minus 10 minutes for every method. KHGT does not publish Imsak or Dhuha, so for `muhammadiyah` those two times follow the Kemenag convention.

`GET /v1/methods` lists every parameter.

### Validating accuracy

`cmd/compare` compares OpenPray's output with an official schedule and prints how far each prayer time is off:

```sh
# Kemenag, through api.myquran.com (whole months, per city)
go run ./cmd/compare -source kemenag -year 2026 -months 1,4,7,10

# Muhammadiyah, through khgt.muhammadiyah.or.id (per day, exact coordinates)
go run ./cmd/compare -source muhammadiyah -year 2026 -months 9,10 -days $(seq -s, 1 31)
```

You can override method parameters with `-rise`, `-ihtiyat` and `-adjust dhuhr=1,sunrise=-1`. Responses are cached in your user cache directory.

The KHGT endpoint only serves the current month, so data for past months only exists in the cache of whoever fetched it. That is why the Muhammadiyah calibration currently covers September and October 2026. Running the comparison in other months and sharing the results is a very welcome contribution.

## Project layout

```
prayer/            calculation library (public API, zero dependencies)
api/               OpenAPI spec
cmd/server/        HTTP API
cmd/openpray/      CLI
cmd/compare/       validation against official schedules
tools/gencities/   regenerates the embedded city data
internal/
  router/          the single place where endpoints are registered
  handler/         request parsing & response formatting
  service/         business logic
  repository/      data access (calculation methods, cities)
  model/           domain types shared across layers
  middleware/
```

To add an endpoint:
1. Write a handler.
2. Write the service logic.
3. Register the route in `internal/router/router.go`.
4. Document it in `api/openapi.yaml`. A test fails if a route is not in the spec.

## Contributing

Pull requests are welcome. Before you submit one, run:

```sh
gofmt -l . && go vet ./... && go test ./...
```

If you change a method's parameters, include the `go run ./cmd/compare` output from before and after the change.

Issues and pull requests are welcome in English or Indonesian.

## License

Copyright 2026 Reza Afri Suhangga Hasibuan. Licensed under [Apache-2.0](LICENSE); see [NOTICE](NOTICE).
