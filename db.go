package daygo

import "embed"

type Database interface {
	Close() error
	Migrate(embed.FS) error
}
