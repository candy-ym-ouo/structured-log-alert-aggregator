//go:build !mysql

package store

import (
	"context"
	"errors"
)

// NewMySQL is enabled with `-tags mysql` after installing go-sql-driver/mysql.
func NewMySQL(context.Context, string) (*MySQL, error) {
	return nil, errors.New("mysql support requires build tag mysql and go-sql-driver/mysql")
}

type MySQL struct{ *Memory }
