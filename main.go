package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	connStr       string
	selGenericSQL string
	reqFlags      = []string{"dsn", "sql"}
)

func init() {
	flag.StringVar(&connStr, "dsn", "", `-dsn "user=user password=password host=127.0.0.1 port=5423 dbname=name sslmode=disable"`)
	flag.StringVar(&selGenericSQL, "sql", "", `-sql "select '(42,foo)'::type_name"`)
}

func makeRequiredFlagMap(req []string) map[string]bool {
	m := make(map[string]bool)
	for _, flName := range req {
		m[flName] = false
	}

	return m
}

func checkFlags() error {
	reqFlMap := makeRequiredFlagMap(reqFlags)

	flag.Visit(func(fl *flag.Flag) {
		reqFlMap[fl.Name] = true
	})

	notSetup := make([]string, 0)
	for fl, ok := range reqFlMap {
		if !ok {
			notSetup = append(notSetup, fl)
		}
	}

	if len(notSetup) > 0 {
		return fmt.Errorf("Expected required flags `%s`. Check -help", strings.Join(notSetup, ", "))
	}

	return nil
}

func main() {
	flag.Parse()
	if err := checkFlags(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.Connect(ctx, connStr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer pool.Close()

	genericBinary := &pgtype.GenericBinary{}
	if err := pool.QueryRow(
		ctx,
		selGenericSQL,
		pgx.QueryResultFormats{pgx.BinaryFormatCode},
	).Scan(genericBinary); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Binary: %v\n", genericBinary.Bytes)

	rawNumOfFields := genericBinary.Bytes[:4]
	numOfFieldsUint32 := binary.BigEndian.Uint32(rawNumOfFields)
	fmt.Printf(
		"Num of fields: %v (%d)\n",
		rawNumOfFields,
		numOfFieldsUint32,
	)
	ci := pgtype.NewConnInfo()

	// The first 4 bytes are the number of fields - 2
	// Then for each field:
	// 4 bytes - OID
	// 4 bytes - length of value
	// x bytes - value
	typeStart := 4
	for i := uint32(0); i < numOfFieldsUint32; i++ {
		rawOID := genericBinary.Bytes[typeStart : typeStart+4]
		oid := binary.BigEndian.Uint32(rawOID)
		rawValLen := genericBinary.Bytes[typeStart+4 : typeStart+8]
		valLen := binary.BigEndian.Uint32(rawValLen)
		rawVal := genericBinary.Bytes[typeStart+8 : typeStart+8+int(valLen)]
		typeStart += int(valLen) + 8

		var scanned interface{}
		var typeName string

		switch oid {
		case pgtype.Int4OID:
			typeName = "int4 (int32)"
			var dst int32
			if err := ci.Scan(oid, pgtype.BinaryFormatCode, rawVal, &dst); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			scanned = dst

		case pgtype.Float8OID:
			typeName = "float8 (double)"
			var dst float64
			if err := ci.Scan(oid, pgtype.BinaryFormatCode, rawVal, &dst); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			scanned = dst

		case pgtype.TextOID:
			typeName = "text"
			var dst string
			if err := ci.Scan(oid, pgtype.BinaryFormatCode, rawVal, &dst); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			scanned = dst

		default:
			typeName = "Unsupported"
			scanned = "Unsupported"
		}

		fmt.Printf(
			`Field %d:
	OID: %d
	Type %s
	Length: %d
	Value: %v
`,
			i,
			oid,
			typeName,
			valLen,
			scanned,
		)
	}

	os.Exit(0)
}
