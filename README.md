# Postgres binary tester

## Example

```sh
$ tester \
    -dsn "user=admin password=admin host=localhost port=5432 dbname=admin sslmode=disable" \
    -sql "SELECT '(0,0)'::point;"

# Binary: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
# Num of fields: [0 0 0 0] (0)
```
