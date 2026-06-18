package plugin

type CatalogAuditEvent struct {
	PluginID    string
	Action      string
	Result      string
	Permissions int
	Menus       int
}

type CatalogAuditSink interface {
	RecordPluginCatalogEvent(event CatalogAuditEvent) error
}
