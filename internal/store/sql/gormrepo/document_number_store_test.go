package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/repository"
	documentnumbersvc "github.com/tinboxw/skoll/internal/service/documentnumber"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestDocumentNumberStoreTransactionIdempotencyRollbackAndRestart(t *testing.T) {
	db := TestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	uow := storesql.NewUnitOfWorkWithDB(db)
	service := documentnumbersvc.NewService(NewDocumentNumberStore(db))
	input := documentNumberTestInput("tenant-a", "purchase_order", time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), "order-1")

	preview, err := service.Preview(context.Background(), "medical_oa", input)
	if err != nil || preview.Sequence != 1 || preview.Number != "PO-202607-000001" {
		t.Fatalf("initial preview=%+v err=%v", preview, err)
	}
	if _, err = service.Issue(context.Background(), "medical_oa", input); !errors.Is(err, documentnumbersvc.ErrTransactionRequired) {
		t.Fatalf("issue outside transaction error=%v", err)
	}

	first, err := issueDocumentNumber(t, uow, service, input, nil)
	if err != nil || first.Sequence != 1 || first.Duplicate {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	duplicate, err := issueDocumentNumber(t, uow, service, input, nil)
	if err != nil || duplicate.Number != first.Number || !duplicate.Duplicate {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}

	conflictInput := input
	conflictInput.Rule.Prefix = "PX"
	if _, err = issueDocumentNumber(t, uow, service, conflictInput, nil); !errors.Is(err, documentnumbersvc.ErrConflict) {
		t.Fatalf("idempotency conflict error=%v", err)
	}
	conflictInput.IdempotencyKey = "order-new-rule"
	if _, err = issueDocumentNumber(t, uow, service, conflictInput, nil); !errors.Is(err, documentnumbersvc.ErrRuleConflict) {
		t.Fatalf("active rule conflict error=%v", err)
	}
	if _, err = service.Preview(context.Background(), "medical_oa", conflictInput); !errors.Is(err, documentnumbersvc.ErrRuleConflict) {
		t.Fatalf("preview active rule conflict error=%v", err)
	}

	rollbackInput := input
	rollbackInput.IdempotencyKey = "order-2"
	rolledBack, err := issueDocumentNumber(t, uow, service, rollbackInput, errors.New("document create failed"))
	if err == nil || rolledBack.Sequence != 2 {
		t.Fatalf("rollback result=%+v err=%v", rolledBack, err)
	}
	reusedInput := input
	reusedInput.IdempotencyKey = "order-3"
	reused, err := issueDocumentNumber(t, uow, service, reusedInput, nil)
	if err != nil || reused.Sequence != 2 || reused.Number != rolledBack.Number {
		t.Fatalf("reused=%+v rolledBack=%+v err=%v", reused, rolledBack, err)
	}

	restarted := documentnumbersvc.NewService(NewDocumentNumberStore(db))
	preview, err = restarted.Preview(context.Background(), "medical_oa", input)
	if err != nil || preview.Sequence != 3 {
		t.Fatalf("restart preview=%+v err=%v", preview, err)
	}
}

func TestDocumentNumberStoreIsolatesTenantTypeAndPeriod(t *testing.T) {
	db := TestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	uow := storesql.NewUnitOfWorkWithDB(db)
	service := documentnumbersvc.NewService(NewDocumentNumberStore(db))

	inputs := []pluginsdk.DocumentNumberInput{
		documentNumberTestInput("tenant-a", "purchase_order", time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), "a-po-jul"),
		documentNumberTestInput("tenant-b", "purchase_order", time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), "b-po-jul"),
		documentNumberTestInput("tenant-a", "sales_order", time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), "a-so-jul"),
		documentNumberTestInput("tenant-a", "purchase_order", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "a-po-aug"),
	}
	for _, input := range inputs {
		result, err := issueDocumentNumber(t, uow, service, input, nil)
		if err != nil || result.Sequence != 1 {
			t.Fatalf("input=%+v result=%+v err=%v", input, result, err)
		}
	}
}

func TestDocumentNumberStoreIssuesUniqueNumbersConcurrently(t *testing.T) {
	db := TestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	uow := storesql.NewUnitOfWorkWithDB(db)
	service := documentnumbersvc.NewService(NewDocumentNumberStore(db))

	const workers = 32
	results := make(chan pluginsdk.DocumentNumberResult, workers)
	errorsCh := make(chan error, workers)
	var group sync.WaitGroup
	for index := range workers {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			input := documentNumberTestInput("tenant-a", "purchase_order", time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), fmt.Sprintf("order-%02d", index))
			result, err := issueDocumentNumber(t, uow, service, input, nil)
			if err != nil {
				errorsCh <- err
				return
			}
			results <- result
		}(index)
	}
	group.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent issue: %v", err)
	}
	sequences := make([]int, 0, workers)
	numbers := make(map[string]struct{}, workers)
	for result := range results {
		sequences = append(sequences, int(result.Sequence))
		numbers[result.Number] = struct{}{}
	}
	sort.Ints(sequences)
	if len(numbers) != workers || len(sequences) != workers {
		t.Fatalf("numbers=%d sequences=%d", len(numbers), len(sequences))
	}
	for index, sequence := range sequences {
		if sequence != index+1 {
			t.Fatalf("sequences=%v", sequences)
		}
	}
}

func issueDocumentNumber(t *testing.T, uow repository.UnitOfWork, service *documentnumbersvc.Service, input pluginsdk.DocumentNumberInput, callbackErr error) (pluginsdk.DocumentNumberResult, error) {
	t.Helper()
	var result pluginsdk.DocumentNumberResult
	err := uow.Do(context.Background(), func(tx repository.Tx) error {
		issued, issueErr := service.Issue(tx.Context(), "medical_oa", input)
		result = issued
		if issueErr != nil {
			return issueErr
		}
		return callbackErr
	})
	return result, err
}

func documentNumberTestInput(tenantID, documentType string, occurredAt time.Time, idempotencyKey string) pluginsdk.DocumentNumberInput {
	prefix := "PO"
	if documentType == "sales_order" {
		prefix = "SO"
	}
	return pluginsdk.DocumentNumberInput{
		Rule: pluginsdk.DocumentNumberRule{
			DocumentType: documentType, Prefix: prefix, Separator: "-", Period: pluginsdk.DocumentNumberPeriodMonth,
			Width: 6, Start: 1, GapPolicy: pluginsdk.DocumentNumberGapTransactional,
		},
		TenantID: tenantID, Permission: pluginsdk.Permission{Resource: "medical_oa." + documentType, Action: "issue"},
		OccurredAt: occurredAt, IdempotencyKey: idempotencyKey,
	}
}
