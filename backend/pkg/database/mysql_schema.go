package database

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// mysqlIndexedStringSize is the length given to a string column that is indexed but has
// no size of its own, on MySQL. It holds an email address (at most 254 characters), and
// 255 utf8mb4 characters (1020 bytes) fit well inside InnoDB's 3072-byte index limit.
const mysqlIndexedStringSize = 255

// sizeIndexedStrings gives every indexed string column without a size a length, on MySQL
// only. GORM's MySQL driver makes a string column varchar only when its tag says `index`
// or `unique`; a column tagged `uniqueIndex` becomes longtext, and MySQL refuses to index
// it ("Error 1170: BLOB/TEXT column used in key specification without a key length"), so
// a fresh MySQL database could never be migrated. Setting the size here, before the first
// migration, fixes that without touching the model tags: PostgreSQL and SQLite keep their
// existing `text` columns, and no MySQL database can already hold the old columns because
// creating them always failed. Columns with an explicit size or an index prefix length,
// and columns that are not indexed (descriptions, comments), are left alone.
//
// The size is set on GORM's cached schema, which the migrator reads, so this must run
// before the first AutoMigrate on db.
func sizeIndexedStrings(db *gorm.DB, models ...any) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	for _, m := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return err
		}
		for _, idx := range stmt.Schema.ParseIndexes() {
			for _, opt := range idx.Fields {
				f := opt.Field
				if f != nil && f.DataType == schema.String && f.Size == 0 && opt.Length == 0 {
					f.Size = mysqlIndexedStringSize
				}
			}
		}
	}
	return nil
}
