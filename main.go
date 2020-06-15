package main

import (
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	connStr           string
	selGenericSQL     string
	reqFlags          = []string{"dsn", "sql"}
	ci                = pgtype.NewConnInfo()
	errUnsupportedOID = errors.New("unsupported OID")
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

// getRaw returns a raw bytes of value and a new type start.
func getRaw(bytes []byte, start, len uint32) ([]byte, uint32) {
	if len == 0xFFFFFFFF {
		len = 0
	}

	return bytes[start+8 : start+8+len], start + len + 8
}

// scanRaw returns the value and a name of the type.
func scanRaw(oid uint32, bytes []byte, len uint32) (value interface{}, name string, err error) {
	dt, ok := ci.DataTypeForOID(oid)
	if !ok {
		return nil, "", errUnsupportedOID
	}
	name = dt.Name

	if len == 0xFFFFFFFF {
		value = "(NULL)"
	} else {
		if bd, ok := dt.Value.(pgtype.BinaryDecoder); ok {
			if err := bd.DecodeBinary(ci, bytes); err != nil {
				return nil, "", err
			}
		} else {
			return nil, "", fmt.Errorf("%T is not binary pgtype.BinaryDecoder", dt.Value)
		}

		value = dt.Value.Get()
		if value == "(NULL)" {
			value = `"(NULL)"`
		}
	}

	return
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

	genericText := &pgtype.GenericText{}
	if err := pool.QueryRow(
		ctx,
		selGenericSQL,
		pgx.QueryResultFormats{pgx.TextFormatCode},
	).Scan(genericText); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	genericBinary := &pgtype.GenericBinary{}
	if err := pool.QueryRow(
		ctx,
		selGenericSQL,
		pgx.QueryResultFormats{pgx.BinaryFormatCode},
	).Scan(genericBinary); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Text: %v\n", genericText.String)
	fmt.Printf("Binary: %v\n", genericBinary.Bytes)

	rawNumOfFields := genericBinary.Bytes[:4]
	numOfFieldsUint32 := binary.BigEndian.Uint32(rawNumOfFields)
	fmt.Printf(
		"Num of fields: %v (%d)\n",
		rawNumOfFields,
		numOfFieldsUint32,
	)

	// The first 4 bytes are the number of fields - 2
	// Then for each field:
	// 4 bytes - OID
	// 4 bytes - length of value
	// x bytes - value
	typeStart := uint32(4)
	for i := uint32(0); i < numOfFieldsUint32; i++ {
		rawOID := genericBinary.Bytes[typeStart : typeStart+4]
		oid := binary.BigEndian.Uint32(rawOID)

		rawValLen := genericBinary.Bytes[typeStart+4 : typeStart+8]
		valLen := binary.BigEndian.Uint32(rawValLen)

		var rawVal []byte
		rawVal, typeStart = getRaw(genericBinary.Bytes, typeStart, valLen)

		value, typeName, err := scanRaw(oid, rawVal, valLen)
		if err != nil {
			if !errors.Is(err, errUnsupportedOID) {
				fmt.Println(err)
				os.Exit(1)
			}

			value, typeName = "Unsupported", "Unsupported"
		}

		fmt.Printf(
			`Field %d:
	OID: %d (%v)
	Type %s
	Length: %d (%v)
	Value: %v (%v)
`,
			i,
			oid, rawOID,
			typeName,
			valLen, rawValLen,
			value, rawVal,
		)
	}

	os.Exit(0)
}
