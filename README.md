# Postgres binary tester

This tool allows you to extract the binary representation of data types from PostgreSQL.

It is intended to support the development of custom binary serializers and deserializers for PostgreSQL data types (including custom type definitions).

## Example

```sh
$ tester \
    -dsn "user=admin password=admin host=localhost port=5432 dbname=admin sslmode=disable" \
    -sql "SELECT '(0,0)'::point;"

# Binary: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
# Num of fields: [0 0 0 0] (0)
```
