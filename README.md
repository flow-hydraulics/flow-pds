Dev environment

    cp env.example .env
    cp env.example .env.test

    # If docker-compose installed
    make dev

Test

    # Standalone (can NOT have emulator running in docker)
    ./tests-with-emulator.sh

    # With docker-compose environment ("make dev" above)
    go test -v
