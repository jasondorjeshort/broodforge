// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"fmt"

	"zombiezen.com/go/sqlite"
)

// ValueKind is the type of a bound parameter.
type ValueKind int

const (
	KindNull ValueKind = iota
	KindText
	KindInt
	KindReal
)

// Value is one bound parameter: a tagged struct, so callers outside the store never see the
// SQLite binding's types and no parameter is an `any`.
type Value struct {
	Kind ValueKind
	Text *string
	Int  *int64
	Real *float64
}

// Text binds a string.
func Text(s string) Value { return Value{Kind: KindText, Text: &s} }

// Int binds an integer.
func Int(i int64) Value { return Value{Kind: KindInt, Int: &i} }

// Real binds a floating-point number.
func Real(f float64) Value { return Value{Kind: KindReal, Real: &f} }

// Null binds NULL.
func Null() Value { return Value{Kind: KindNull} }

// OptText binds s, or NULL when s is empty.
func OptText(s string) Value {
	if s == "" {
		return Null()
	}
	return Text(s)
}

// Row is the current row of a query, read by column index.
type Row struct{ stmt *sqlite.Stmt }

// Text is column i as text; NULL is "".
func (r *Row) Text(i int) string { return r.stmt.ColumnText(i) }

// Int is column i as an integer; NULL is 0.
func (r *Row) Int(i int) int64 { return r.stmt.ColumnInt64(i) }

// Float is column i as a floating-point number; NULL is 0.
func (r *Row) Float(i int) float64 { return r.stmt.ColumnFloat(i) }

// IsNull reports whether column i is NULL.
func (r *Row) IsNull(i int) bool { return r.stmt.ColumnIsNull(i) }

// query1 prepares one statement, binds args, steps it to the end and calls row for each row
// (row may be nil). The statement is cached on the connection.
func query1(conn *sqlite.Conn, query string, args []Value, row func(*Row) error) (err error) {
	stmt, err := conn.Prepare(query)
	if err != nil {
		return err
	}
	defer func() {
		if rerr := stmt.Reset(); err == nil {
			err = rerr
		}
		if cerr := stmt.ClearBindings(); err == nil {
			err = cerr
		}
	}()
	if n := stmt.BindParamCount(); n != len(args) {
		return fmt.Errorf("query wants %d parameters, got %d: %s", n, len(args), query)
	}
	for i, a := range args {
		switch a.Kind {
		case KindNull:
			stmt.BindNull(i + 1)
		case KindText:
			stmt.BindText(i+1, *a.Text)
		case KindInt:
			stmt.BindInt64(i+1, *a.Int)
		case KindReal:
			stmt.BindFloat(i+1, *a.Real)
		}
	}
	r := &Row{stmt: stmt}
	for {
		more, err := stmt.Step()
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
		if row != nil {
			if err := row(r); err != nil {
				return err
			}
		}
	}
}
