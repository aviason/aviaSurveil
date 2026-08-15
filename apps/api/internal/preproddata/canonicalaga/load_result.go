package canonicalaga

// LoadResult is the deterministic receipt returned by an approved catalog
// import. It intentionally contains counts and digests only; it is not an
// approval or publication decision.
type LoadResult struct {
	CatalogID      string
	CatalogVersion string
	QuestionCount  int
	FormCount      int
	ImportDigest   string
}
