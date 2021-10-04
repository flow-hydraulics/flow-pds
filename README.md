# Flow Pack Distribution Service

## Running the PDS backend service

Docker

    cp env.example .env # edit according to your setup
    docker run -p 3000:3000 --env-file .env ghcr.io/flow-hydraulics/flow-pds:latest

Dev environment

    cp env.example .env

    # Needs docker-compose installed
    make dev

## Testing

    cp env.example .env.test

    # Standalone (can NOT have emulator running in docker)
    ./tests-with-emulator.sh

    # With docker-compose environment ("make dev" above)
    go test -v


## Project layout

Backend service code: `./service`

Contract code (test, deploy): `./go-contracts`

API spec:
- `./models`
- `./reference`

Simple API tests: `./api-scripts`

Cadence source code:
- `./cadence-contracts`
- `./cadence-scripts`
- `./cadence-transactions`

## Configuration

### Database

| Config variable | Environment variable        | Description                                                                                      | Default     | Examples                  |
| --------------- | :-------------------------- | ------------------------------------------------------------------------------------------------ | ----------- | ------------------------- |
| DatabaseType    | `FLOW_PDS_DATABASE_DSN` | Type of database driver                                                                          | `sqlite`    | `sqlite`, `psql`, `mysql` |
| DatabaseDSN     | `FLOW_PDS_DATABASE_TYPE`  | Data source name ([DSN](https://en.wikipedia.org/wiki/Data_source_name)) for database connection | `pds.db` | See below                 |

Examples of Database DSN

    mysql://john:pass@localhost:3306/my_db

    postgresql://postgres:postgres@localhost:5432/postgres

    user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local

    host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai

For more: https://gorm.io/docs/connecting_to_the_database.html


### All possible configuration variables

Refer to [service/config/config.go](service/config/config.go) for details and documentation.
