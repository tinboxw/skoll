package pluginsdk

import "testing"

func TestParseHostCapabilityAcceptsOnlyCurrentExactOperations(t *testing.T) {
	got, err := ParseHostCapability("  DATASTORE.QUERY ")
	if err != nil {
		t.Fatalf("ParseHostCapability error: %v", err)
	}
	if got != HostCapabilityDatastoreQuery {
		t.Fatalf("ParseHostCapability=%q want=%q", got, HostCapabilityDatastoreQuery)
	}

	for _, value := range []string{"", "datastore.*", "*", "secrets", "lifecycle.uninstall", "legacy.datastore.query"} {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseHostCapability(value); err == nil {
				t.Fatalf("expected %q to be rejected", value)
			}
		})
	}
}
