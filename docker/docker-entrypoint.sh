#!/bin/sh
set -e

ENV_FILE="/app/.env"
EXAMPLE_FILE="/app/.env.example"

# If no explicit env file was baked or mounted, fall back to the example.
if [ ! -f "$ENV_FILE" ] && [ -f "$EXAMPLE_FILE" ]; then
    cp "$EXAMPLE_FILE" "$ENV_FILE"
fi

# Load variables from the env file only when they are not already provided
# (allowing ECS/Compose/`docker run --env` to take precedence).
if [ -f "$ENV_FILE" ]; then
    while IFS='=' read -r key value; do
        case "$key" in
            ''|'#'* ) continue ;;
        esac
        # Strip carriage returns for Windows-formatted files
        value="$(printf '%s' "$value" | tr -d '\r')"
        if eval "[ -z \"\${$key+x}\" ]"; then
            export "$key=$value"
        fi
    done < "$ENV_FILE"
fi

exec "$@"