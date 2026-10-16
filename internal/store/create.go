// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"os"
	"path/filepath"
	"time"
)

// Create creates data/store.sqlite under root if it is missing, applies every migration it
// lacks and returns the writer, with the audit authorizer set. It is the bot's start step 4
// (03-architecture.md, Startup) and what tests and scratch roots use to make a store; nothing
// else creates or migrates one. A store holding a migration this binary does not know is
// refused with store-schema (ADR 0021).
func Create(root string) (*Store, error) {
	return create(root, time.Now())
}

func create(root string, now time.Time) (*Store, error) {
	path := Path(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	conn, err := openConn(path)
	if err != nil {
		return nil, err
	}
	if err := migrate(conn, now); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.SetAuthorizer(auditGuard); err != nil {
		conn.Close()
		return nil, err
	}
	return &Store{conn: conn, path: path}, nil
}
