package deadletter

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// fakeConsumer is a tablerow.Consumer whose Consume result we control.
type fakeConsumer struct {
	err     error
	calls   int
	lastSQL string
}

func (f *fakeConsumer) Consume(query string) error {
	f.calls++
	f.lastSQL = query
	return f.err
}

func TestConsume_PassThroughBeforeArm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	inner := &fakeConsumer{err: errors.New("data too long")}
	w := New(inner, db)

	// Not armed: the underlying error must surface unchanged, and nothing is
	// written to the deadletter table.
	if err := w.Consume("UPDATE t SET x=1"); err == nil {
		t.Fatal("expected error to pass through before Arm")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB activity before Arm: %v", err)
	}
}

func TestConsume_SuccessIsPassThrough(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	inner := &fakeConsumer{err: nil}
	w := New(inner, db)
	w.Arm()

	if err := w.Consume("UPDATE t SET x=1"); err != nil {
		t.Fatalf("expected nil on success, got %v", err)
	}
	// A successful statement must never touch the deadletter table.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB activity on success: %v", err)
	}
}

func TestConsume_CapturesAfterArm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	inner := &fakeConsumer{err: errors.New("Error 1406: Data too long")}
	w := New(inner, db)
	w.Arm()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO deadletter (query, error) VALUES (?, ?)")).
		WithArgs("UPDATE t SET x=1", "Error 1406: Data too long").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Armed: the failure is captured and Consume returns nil so the loop lives.
	if err := w.Consume("UPDATE t SET x=1"); err != nil {
		t.Fatalf("expected nil after capture, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected capture insert: %v", err)
	}
}

func TestConsume_CaptureFailureIsNeverFatal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	inner := &fakeConsumer{err: errors.New("boom")}
	w := New(inner, db)
	w.Arm()

	// Even if recording the dead-letter fails, Consume must return nil.
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO deadletter")).
		WillReturnError(errors.New("db down"))

	if err := w.Consume("UPDATE t SET x=1"); err != nil {
		t.Fatalf("expected nil even when capture fails, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected capture attempt: %v", err)
	}
}

func TestReset(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	w := New(&fakeConsumer{}, db)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM deadletter")).
		WillReturnResult(sqlmock.NewResult(0, 3))

	if err := w.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected delete: %v", err)
	}
}

func TestCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	w := New(&fakeConsumer{}, db)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM deadletter")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

	n, err := w.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 7 {
		t.Fatalf("expected count 7, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected count query: %v", err)
	}
}
