# load .env DB_URL
source .env

# run the migration
goose -dir "./sql/schema" postgres "$DB_URL" "$@"