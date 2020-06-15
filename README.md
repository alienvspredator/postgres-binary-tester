# Postgres binary tester

## Example

```sh
go run main.go -dsn "user=admin password=admin host=localhost port=5432 dbname=admin sslmode=disable" -sql "SELECT '(0,0)'::point;"
```
