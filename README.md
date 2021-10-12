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
| DatabaseType    | `FLOW_PDS_DATABASE_DSN`     | Type of database driver                                                                          | `sqlite`    | `sqlite`, `psql`, `mysql` |
| DatabaseDSN     | `FLOW_PDS_DATABASE_TYPE`    | Data source name ([DSN](https://en.wikipedia.org/wiki/Data_source_name)) for database connection | `pds.db`    | See below                 |

Examples of Database DSN

    mysql://john:pass@localhost:3306/my_db

    postgresql://postgres:postgres@localhost:5432/postgres

    user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local

    host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai

For more: https://gorm.io/docs/connecting_to_the_database.html


### Google KMS admin key

In order to use a key stored in Google KMS as admin key:
- first create the key in Google KMS
- export the public key & resource name
- convert the key using flow-cli (`flow keys decode pem --from-file kms-key-export.pem`)
- add the key to the PDS account (for testing in emulator; `flow transactions send ./cadence-transactions/keys/add-key.cdc <public key> --signer emulator-pds`)
  - when testing locally the added key will usually be in index 1, remember to update `FLOW_PDS_ADMIN_PRIVATE_KEY_INDEXES` accordingly
- modify the following configuration settings:

| Config variable | Environment variable | Description | Default | Examples |
| --- | :-- | --- | --- | --- |
| AdminPrivateKey | `FLOW_PDS_ADMIN_PRIVATE_KEY` | Private key value, for Google KMS this should be the Resource Name of the key | `""` | `projects/KMS_PROJECT_NAME/locations/KMS_PROJECT_LOCATION/keyRings/KMS_KEYRING_NAME/cryptoKeys/KMS_ADMIN_KEY_NAME/cryptoKeyVersions/1`, `9c687961e7a1abe1e445830e7ec118ffd1e2a0449cf705f5476b3f100e94dc29` |
| AdminPrivateKeyIndexes | `FLOW_PDS_ADMIN_PRIVATE_KEY_INDEXES` | Comma separated list of key indexes that can be used. | `0` | `1,2,3` |
| AdminPrivateKeyType | `FLOW_PDS_ADMIN_PRIVATE_KEY_TYPE` | Type of key, `google_kms` for Google KMS | `local` | `local`, `google_kms` |
| - | `GOOGLE_APPLICATION_CREDENTIALS` | Path the the Google KMS credentials JSON file. |  | `/path/to/kms-credentials.json` |



### All possible configuration variables

Refer to [service/config/config.go](service/config/config.go) for details and documentation.
