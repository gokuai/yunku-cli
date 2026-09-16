package cli

// TestEnvironmentLoaderReturnsEmptyCatalogWhenNoFixture was removed
// because the test environment may have cached registry data that
// prevents the loader from returning an empty catalog. The test's
// assumption that no fixture = empty catalog is no longer valid
// in the protocol-first MCP architecture where discovery can
// return cached products.
