A provider directory holding a resource schema and no source, which is how
equinix appears in the real tree. The extractor has to say the connector is
missing for a reason rather than leave it out silently.

The marker is `resources/` rather than `dist/` because the repository's
.gitignore drops every `dist/`, and a fixture directory that never gets
committed makes the test that depends on it pass everywhere except here.
