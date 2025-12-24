package database

import "errors"

var ErrOptimisticLock = errors.New("document modified by another process")
var ErrNotFound = errors.New("document not found")
var ErrDocumentExists = errors.New("document already exists")
