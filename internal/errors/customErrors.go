package customErrors

import "errors"

var (
	ErrGetNotifyList  error = errors.New("can't execute select query")
	ErrFoundToken     error = errors.New("can't find bot token")
	ErrInitPostgreSQL error = errors.New("Failed init postgreSQL")
	ErrDBConn         error = errors.New("failed to create database connection")
	ErrDBPing         error = errors.New("failed ping the database")

	ErrEmptyRow         error = errors.New("BD has no rows")
	ErrBookingIntersect error = errors.New("booking intersect")
)
