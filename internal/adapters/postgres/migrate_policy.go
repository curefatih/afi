package postgres

// schemaDecision captures what Migrate inspected in the database.
type schemaDecision struct {
	MetaExists    bool
	MetaVersion   int
	MetaVersionOK bool // true when MetaVersion was read successfully
	OrgExists     bool
	OrgIDDataType string
}

// shouldWipeSchema returns true only for one-time legacy/incomplete installs.
// A schema version mismatch never wipes application data.
func shouldWipeSchema(d schemaDecision) bool {
	if d.MetaExists && d.MetaVersionOK {
		return false
	}
	if !d.OrgExists {
		return false
	}
	// Legacy UUID schema or text schema without afi_schema_meta.
	return true
}
