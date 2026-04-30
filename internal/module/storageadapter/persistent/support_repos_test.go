package persistent_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/module/fileservice"
	"github.com/tinboxw/skoll/internal/module/modgenerator"
	"github.com/tinboxw/skoll/internal/module/releasegov"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

// --- generator -------------------------------------------------------------

func TestGeneratorRepository_PassThrough(t *testing.T) {
	repo := persistent.NewGeneratorRepository()
	res, err := repo.Generate("user")
	if err != nil || res.Module != "user" || len(res.Artifacts) == 0 {
		t.Fatalf("generate: %+v err=%v", res, err)
	}
	res2, err := repo.GenerateWithSchema("user", &modgenerator.FormSchema{Fields: []modgenerator.FormField{{Name: "x"}}}, "v2")
	if err != nil {
		t.Fatalf("generate-with-schema: %v", err)
	}
	if res2.Module != "user" {
		t.Fatalf("schema: %+v", res2)
	}
}

// --- release ---------------------------------------------------------------

func TestReleaseRepository_SubmitAndScorecard(t *testing.T) {
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.ReleaseRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewReleaseRepository(gdb)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	if _, err := repo.SubmitEvidence(releasegov.EvidenceInput{}, now); !errors.Is(err, releasegov.ErrMilestoneRequired) {
		t.Fatalf("expected milestone-required, got %v", err)
	}
	if _, err := repo.SubmitEvidence(releasegov.EvidenceInput{
		Milestone:        "m1",
		GoTestPassed:     true,
		GoRacePassed:     true,
		ReadmeSynced:     true,
		BenchmarkNsPerOp: 90,
		BaselineNsPerOp:  100,
	}, now); err != nil {
		t.Fatalf("submit: %v", err)
	}

	card := repo.Scorecard("M1", 0.1)
	if !card.QualityGatePassed || !card.ReleaseReady || card.EvidenceCount != 1 {
		t.Fatalf("scorecard: %+v", card)
	}
}

func TestReleaseRepository_HydrateAcrossInstances(t *testing.T) {
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.ReleaseRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewReleaseRepository(gdb)
	if err != nil {
		t.Fatalf("first new: %v", err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	if _, err := repo.SubmitEvidence(releasegov.EvidenceInput{
		Milestone: "m1", GoTestPassed: true, GoRacePassed: true, ReadmeSynced: true,
	}, now); err != nil {
		t.Fatalf("submit: %v", err)
	}

	repo2, err := persistent.NewReleaseRepository(gdb)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	card := repo2.Scorecard("m1", 0.1)
	if card.EvidenceCount != 1 || !card.QualityGatePassed {
		t.Fatalf("hydrated scorecard: %+v", card)
	}
}

// --- file ------------------------------------------------------------------

func newFileRepo(t *testing.T) *persistent.FileRepository {
	t.Helper()
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.FileRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	root := t.TempDir()
	backend, err := fileservice.NewLocalBackend(root)
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	repo, err := persistent.NewFileRepository(gdb, backend)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	t.Cleanup(func() { _ = filepath.Clean(root) })
	return repo
}

func TestFileRepository_UploadGetListDownload(t *testing.T) {
	repo := newFileRepo(t)
	f, err := repo.Upload("hello.txt", []byte("hello"))
	if err != nil || f.ID == 0 || f.StorageKey == "" {
		t.Fatalf("upload: %+v err=%v", f, err)
	}

	got, err := repo.Get(f.ID)
	if err != nil || got.Name != "hello.txt" || got.Size != 5 {
		t.Fatalf("get: %+v err=%v", got, err)
	}
	if list := repo.List(); len(list) != 1 {
		t.Fatalf("list: %+v", list)
	}
	meta, body, err := repo.Download(f.ID)
	if err != nil || string(body) != "hello" || meta.ID != f.ID {
		t.Fatalf("download: %+v body=%q err=%v", meta, body, err)
	}
}

func TestFileRepository_HydrateAcrossInstances(t *testing.T) {
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.FileRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	root := t.TempDir()
	backend, err := fileservice.NewLocalBackend(root)
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	repo, err := persistent.NewFileRepository(gdb, backend)
	if err != nil {
		t.Fatalf("first new: %v", err)
	}
	first, err := repo.Upload("a.txt", []byte("aaa"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	repo2, err := persistent.NewFileRepository(gdb, backend)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	got, err := repo2.Get(first.ID)
	if err != nil || got.Size != 3 {
		t.Fatalf("hydrated get: %+v err=%v", got, err)
	}
	_, body, err := repo2.Download(first.ID)
	if err != nil || string(body) != "aaa" {
		t.Fatalf("hydrated download: body=%q err=%v", body, err)
	}
	next, err := repo2.Upload("b.txt", []byte("bb"))
	if err != nil || next.ID <= first.ID {
		t.Fatalf("post-hydrate upload id: %+v err=%v", next, err)
	}
}
