package plugin

type CatalogAuditEvent struct {
	PluginID     string
	Action       string
	Result       string
	Permissions  int
	Menus        int
	Routes       int
	Events       int
	AuditActions int
}

type CatalogAuditSink interface {
	RecordPluginCatalogEvent(event CatalogAuditEvent) error
}
