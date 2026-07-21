package pharmaoa

import (
	"context"
	"strings"
	"testing"

	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type adapterAuditPort struct {
	entry pluginsdk.AuditEntry
}

func (p *adapterAuditPort) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	p.entry = entry
	return pluginsdk.AuditReceipt{ID: "audit-1"}, nil
}

type adapterFilePort struct {
	stored pluginsdk.FileWrite
	item   pluginsdk.FileObject
}

func (p *adapterFilePort) Store(_ context.Context, input pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	p.stored = input
	return p.item, nil
}

func (*adapterFilePort) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return nil, nil
}

func (p *adapterFilePort) Get(context.Context, string) (pluginsdk.FileObject, error) {
	return p.item, nil
}

func (*adapterFilePort) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{}, nil
}

func (*adapterFilePort) Delete(context.Context, string) error { return nil }

func TestHostAuditAdapterUsesBoundedPluginNamespace(t *testing.T) {
	port := &adapterAuditPort{}
	adapter := newHostAuditAdapter(port)
	if _, err := adapter.Append(context.Background(), "claimed-user", "pharma_oa.customer.update", "pharma_oa_customer", "customer-1", map[string]any{"changed": true}); err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if port.entry.Action != "customer.update" || port.entry.Resource != "customer" {
		t.Fatalf("plugin namespace was not normalized: %+v", port.entry)
	}
	if port.entry.Detail["reportedActorId"] != "claimed-user" {
		t.Fatalf("reported actor evidence is missing: %+v", port.entry.Detail)
	}
}

func TestHostFileAdapterUsesPublicFilePort(t *testing.T) {
	port := &adapterFilePort{item: pluginsdk.FileObject{ID: "file-1", Name: "report.csv", Status: "available"}}
	adapter := newHostFileAdapter(port)
	object, err := adapter.Upload(context.Background(), filesvc.UploadInput{
		Key: "reports/report.csv", Name: "report.csv", Body: strings.NewReader("report"), Metadata: map[string]string{"job": "1"},
	})
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
	if object.ID.String() != "file-1" || string(port.stored.Content) != "report" || port.stored.Visibility != pluginsdk.FileVisibilityPrivate {
		t.Fatalf("public file port was not used: object=%+v input=%+v", object, port.stored)
	}
	got, decision, err := adapter.Get(context.Background(), filesvc.GetInput{FileID: "file-1"})
	if err != nil || !decision.Allowed || got.ID.String() != "file-1" {
		t.Fatalf("Get object=%+v decision=%+v err=%v", got, decision, err)
	}
}
