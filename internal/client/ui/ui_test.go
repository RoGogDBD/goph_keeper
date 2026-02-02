package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"goph_keeper/internal/client/api"
	"goph_keeper/internal/client/crypto"
	"goph_keeper/internal/client/store"
)

func TestNewLocalID(t *testing.T) {
	t.Parallel()
	id := newLocalID()
	if len(id) != 32 {
		t.Fatalf("id len=%d want=32", len(id))
	}
}

func TestSyncItemRoundTrip(t *testing.T) {
	t.Parallel()

	item := store.Item{
		ID:        "id1",
		Type:      "text",
		Payload:   []byte("p"),
		Meta:      map[string]string{"a": "b"},
		Deleted:   false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	out := toSyncItems([]store.Item{item})
	if len(out) != 1 {
		t.Fatalf("toSyncItems len=%d", len(out))
	}
	back := fromSyncItems(out)
	if len(back) != 1 || back[0].ID != item.ID {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestGetCrypto(t *testing.T) {
	t.Parallel()

	c, err := crypto.NewCrypto("secret", t.TempDir())
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}
	cli := api.New("http://example.com", nil, c)
	if getCrypto(cli) != c {
		t.Fatalf("getCrypto mismatch")
	}
}

func TestModal(t *testing.T) {
	t.Parallel()
	if modal(nil, 10, 5) == nil {
		t.Fatalf("modal is nil")
	}
}

func TestShowMasterPasswordForm(t *testing.T) {
	t.Parallel()

	app := tview.NewApplication()
	pages := tview.NewPages()
	status := tview.NewTextView()
	state := &uiState{}
	cli := api.New("http://example.com", nil, nil)

	showMasterPasswordForm(app, pages, cli, status, t.TempDir(), state)
	form := mustGetForm(t, pages)
	field := mustInputField(t, form, 0)
	field = field.SetText("secret")
	if field.GetText() != "secret" {
		t.Fatalf("expected secret")
	}
	pressFormButton(form, 0)

	if !state.masterPasswordSet {
		t.Fatalf("master password not set")
	}
	if getCrypto(cli) == nil {
		t.Fatalf("crypto not set on client")
	}
}

func TestShowAddAndGetSecret(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	local, err := store.NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	t.Cleanup(func() {
		if err := local.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	crypt, err := crypto.NewCrypto("secret", dir)
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}
	cli := api.New("http://example.com", nil, crypt)

	app := tview.NewApplication()
	pages := tview.NewPages()
	status := tview.NewTextView()
	state := &uiState{masterPasswordSet: true}

	showAddSecret(app, pages, local, cli, status, state)
	form := mustGetForm(t, pages)
	field := mustInputField(t, form, 0)
	field = field.SetText("text")
	if field.GetText() != "text" {
		t.Fatalf("expected text")
	}
	field = mustInputField(t, form, 1)
	field = field.SetText("payload")
	if field.GetText() != "payload" {
		t.Fatalf("expected payload")
	}
	field = mustInputField(t, form, 2)
	field = field.SetText("k=v")
	if field.GetText() != "k=v" {
		t.Fatalf("expected meta")
	}
	pressFormButton(form, 0)

	items, err := local.List(context.Background(), false)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one item, err=%v len=%d", err, len(items))
	}

	showGetSecret(app, pages, local, cli, status, state)
	form = mustGetForm(t, pages)
	field = mustInputField(t, form, 0)
	field = field.SetText(items[0].ID)
	if field.GetText() != items[0].ID {
		t.Fatalf("expected id")
	}
	pressFormButton(form, 0)
}

func TestShowAuthForm(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/register":
			w.WriteHeader(http.StatusCreated)
		case "/api/login":
			if err := json.NewEncoder(w).Encode(api.LoginResponse{Token: "t"}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	app := tview.NewApplication()
	pages := tview.NewPages()
	status := tview.NewTextView()
	state := &uiState{}
	cli := api.New(ts.URL, &memoryTokenStore{}, nil)

	showAuthForm(app, pages, cli, status, "register", state)
	form := mustGetForm(t, pages)
	field := mustInputField(t, form, 0)
	field = field.SetText("a@b.c")
	if field.GetText() != "a@b.c" {
		t.Fatalf("expected email")
	}
	field = mustInputField(t, form, 1)
	field = field.SetText("p")
	if field.GetText() != "p" {
		t.Fatalf("expected password")
	}
	pressFormButton(form, 0)

	showAuthForm(app, pages, cli, status, "login", state)
	form = mustGetForm(t, pages)
	field = mustInputField(t, form, 0)
	field = field.SetText("a@b.c")
	if field.GetText() != "a@b.c" {
		t.Fatalf("expected email")
	}
	field = mustInputField(t, form, 1)
	field = field.SetText("p")
	if field.GetText() != "p" {
		t.Fatalf("expected password")
	}
	pressFormButton(form, 0)
}

func TestShowListAndDoSync(t *testing.T) {
	t.Parallel()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("screen init: %v", err)
	}

	app := tview.NewApplication().SetScreen(screen)
	pages := tview.NewPages()
	status := tview.NewTextView()
	app.SetRoot(pages, true)

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()
	t.Cleanup(app.Stop)

	dir := t.TempDir()
	local, err := store.NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	t.Cleanup(func() {
		if err := local.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	item := store.Item{
		ID:        "id1",
		Type:      "text",
		Payload:   []byte("payload"),
		Meta:      map[string]string{},
		Deleted:   false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := local.Upsert(context.Background(), item, false); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sync" && r.Method == http.MethodGet {
			if err := json.NewEncoder(w).Encode(api.SyncPullResponse{Items: []api.SyncItem{}}); err != nil {
				t.Fatalf("encode: %v", err)
			}
			return
		}
		if r.URL.Path == "/api/sync" && r.Method == http.MethodPost {
			if err := json.NewEncoder(w).Encode(api.SyncPushResponse{Applied: 0}); err != nil {
				t.Fatalf("encode: %v", err)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)

	cli := api.New(ts.URL, &memoryTokenStore{token: "t"}, nil)

	showList(app, pages, local, status)
	doSync(app, cli, local, status, dir)

	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	default:
	}
}

func mustGetForm(t *testing.T, pages *tview.Pages) *tview.Form {
	t.Helper()
	page := pages.GetPage("modal")
	outer, ok := page.(*tview.Flex)
	if !ok {
		t.Fatalf("modal is not flex")
	}
	inner, ok := outer.GetItem(1).(*tview.Flex)
	if !ok {
		t.Fatalf("inner is not flex")
	}
	form, ok := inner.GetItem(1).(*tview.Form)
	if !ok {
		t.Fatalf("form not found")
	}
	return form
}

func mustInputField(t *testing.T, form *tview.Form, index int) *tview.InputField {
	t.Helper()
	field, ok := form.GetFormItem(index).(*tview.InputField)
	if !ok {
		t.Fatalf("form item %d is not input field", index)
	}
	return field
}

func pressFormButton(form *tview.Form, index int) {
	btn := form.GetButton(index)
	handler := btn.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
}

type memoryTokenStore struct {
	token string
}

func (s *memoryTokenStore) Load() (string, error) {
	return s.token, nil
}

func (s *memoryTokenStore) Save(token string) error {
	s.token = token
	return nil
}
