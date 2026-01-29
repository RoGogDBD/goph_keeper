package ui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"

	"goph_keeper/internal/client/api"
	"goph_keeper/internal/client/crypto"
	"goph_keeper/internal/client/store"
)

func RunUI(baseURL, dataDir string) error {
	app := tview.NewApplication()
	pages := tview.NewPages()
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetText("GophKeeper UI")

	tokenStore := store.NewTokenStore(dataDir)
	cli := api.New(baseURL, tokenStore, nil)
	local, err := store.NewLocalStore(dataDir)
	if err != nil {
		return err
	}
	defer local.Close()

	state := &uiState{}

	main := tview.NewFlex().SetDirection(tview.FlexRow)
	menu := tview.NewList().ShowSecondaryText(false)
	menu.AddItem("Register", "", 0, func() { showAuthForm(app, pages, cli, status, "register", state) })
	menu.AddItem("Login", "", 0, func() { showAuthForm(app, pages, cli, status, "login", state) })
	menu.AddItem("Set Master Password", "", 0, func() { showMasterPasswordForm(app, pages, cli, status, dataDir, state) })
	menu.AddItem("List Secrets", "", 0, func() { showList(app, pages, local, status) })
	menu.AddItem("Add Secret", "", 0, func() { showAddSecret(app, pages, local, cli, status, state) })
	menu.AddItem("Get Secret", "", 0, func() { showGetSecret(app, pages, local, cli, status, state) })
	menu.AddItem("Sync", "", 0, func() { doSync(app, cli, local, status, dataDir) })
	menu.AddItem("Quit", "", 0, func() { app.Stop() })

	main.AddItem(menu, 0, 1, true)
	main.AddItem(status, 1, 0, false)

	pages.AddPage("main", main, true, true)

	return app.SetRoot(pages, true).Run()
}

type uiState struct {
	masterPasswordSet bool
}

func showAuthForm(app *tview.Application, pages *tview.Pages, cli *api.Client, status *tview.TextView, mode string, state *uiState) {
	form := tview.NewForm()
	var email, password string
	form.AddInputField("Email", "", 40, nil, func(v string) { email = v })
	form.AddPasswordField("Password", "", 40, '*', func(v string) { password = v })
	form.AddButton("Submit", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var err error
		if mode == "register" {
			err = cli.Register(ctx, api.RegisterRequest{Email: email, Password: password})
		} else {
			err = cli.Login(ctx, api.LoginRequest{Email: email, Password: password})
		}
		if err != nil {
			status.SetText(fmt.Sprintf("[red]Error: %v", err))
		} else {
			status.SetText("OK")
		}
		pages.RemovePage("modal")
	})
	form.AddButton("Cancel", func() { pages.RemovePage("modal") })
	title := "Login"
	if mode == "register" {
		title = "Register"
	}
	form.SetBorder(true).SetTitle(title)

	pages.AddPage("modal", modal(form, 60, 12), true, true)
	app.SetFocus(form)
}

func showMasterPasswordForm(app *tview.Application, pages *tview.Pages, cli *api.Client, status *tview.TextView, dataDir string, state *uiState) {
	form := tview.NewForm()
	var master string
	form.AddPasswordField("Master Password", "", 40, '*', func(v string) { master = v })
	form.AddButton("Set", func() {
		crypto, err := crypto.NewCrypto(master, dataDir)
		if err != nil {
			status.SetText(fmt.Sprintf("[red]Error: %v", err))
		} else {
			cli.SetCrypto(crypto)
			state.masterPasswordSet = true
			status.SetText("Master password set")
		}
		pages.RemovePage("modal")
	})
	form.AddButton("Cancel", func() { pages.RemovePage("modal") })
	form.SetBorder(true).SetTitle("Master Password")

	pages.AddPage("modal", modal(form, 60, 10), true, true)
	app.SetFocus(form)
}

func showList(app *tview.Application, pages *tview.Pages, local *store.LocalStore, status *tview.TextView) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, err := local.List(ctx, false)
		app.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			list := tview.NewList().ShowSecondaryText(true)
			for _, s := range items {
				secondary := s.UpdatedAt.Format(time.RFC3339)
				list.AddItem(s.ID, s.Type+"  "+secondary, 0, nil)
			}
			list.AddItem("Back", "", 0, func() { pages.RemovePage("modal") })
			list.SetBorder(true).SetTitle("Secrets")
			pages.AddPage("modal", modal(list, 80, 20), true, true)
			app.SetFocus(list)
		})
	}()
}

func showAddSecret(app *tview.Application, pages *tview.Pages, local *store.LocalStore, cli *api.Client, status *tview.TextView, state *uiState) {
	form := tview.NewForm()
	var typ, payload, meta string
	form.AddInputField("Type", "", 30, nil, func(v string) { typ = v })
	form.AddInputField("Payload", "", 30, nil, func(v string) { payload = v })
	form.AddInputField("Meta (k=v,...)", "", 30, nil, func(v string) { meta = v })
	form.AddButton("Add", func() {
		if !state.masterPasswordSet {
			status.SetText("[red]Master password not set")
			pages.RemovePage("modal")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		now := time.Now().UTC()
		item := store.Item{
			ID:        newLocalID(),
			Type:      typ,
			Payload:   []byte(payload),
			Meta:      store.ParseMeta(meta),
			Deleted:   false,
			CreatedAt: now,
			UpdatedAt: now,
		}
		cryptoSvc := getCrypto(cli)
		if cryptoSvc != nil {
			enc, err := cryptoSvc.Encrypt(item.Payload)
			if err != nil {
				status.SetText(fmt.Sprintf("[red]Error: %v", err))
				pages.RemovePage("modal")
				return
			}
			item.Payload = enc
		}
		if err := local.Upsert(ctx, item, true); err != nil {
			status.SetText(fmt.Sprintf("[red]Error: %v", err))
		} else {
			status.SetText(fmt.Sprintf("Created: %s", item.ID))
		}
		pages.RemovePage("modal")
	})
	form.AddButton("Cancel", func() { pages.RemovePage("modal") })
	form.SetBorder(true).SetTitle("Add Secret")

	pages.AddPage("modal", modal(form, 70, 12), true, true)
	app.SetFocus(form)
}

func showGetSecret(app *tview.Application, pages *tview.Pages, local *store.LocalStore, cli *api.Client, status *tview.TextView, state *uiState) {
	form := tview.NewForm()
	var id string
	form.AddInputField("ID", "", 40, nil, func(v string) { id = v })
	form.AddButton("Get", func() {
		if !state.masterPasswordSet {
			status.SetText("[red]Master password not set")
			pages.RemovePage("modal")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		secret, err := local.Get(ctx, id)
		if err != nil {
			status.SetText(fmt.Sprintf("[red]Error: %v", err))
			pages.RemovePage("modal")
			return
		}
		cryptoSvc := getCrypto(cli)
		if cryptoSvc != nil && len(secret.Payload) > 0 {
			dec, err := cryptoSvc.Decrypt(secret.Payload)
			if err != nil {
				status.SetText(fmt.Sprintf("[red]Error: %v", err))
				pages.RemovePage("modal")
				return
			}
			secret.Payload = dec
		}
		info := tview.NewTextView().SetDynamicColors(true)
		info.SetText(fmt.Sprintf("ID: %s\nType: %s\nPayload: %s\nMeta: %v\nUpdated: %s",
			secret.ID, secret.Type, strings.TrimSpace(string(secret.Payload)), secret.Meta, secret.UpdatedAt.Format(time.RFC3339)))
		info.SetBorder(true).SetTitle("Secret")
		pages.RemovePage("modal")
		pages.AddPage("modal", modal(info, 80, 14), true, true)
		app.SetFocus(info)
	})
	form.AddButton("Cancel", func() { pages.RemovePage("modal") })
	form.SetBorder(true).SetTitle("Get Secret")

	pages.AddPage("modal", modal(form, 70, 10), true, true)
	app.SetFocus(form)
}

func doSync(app *tview.Application, cli *api.Client, local *store.LocalStore, status *tview.TextView, dataDir string) {
	go func() {
		syncStore := store.NewSyncStore(dataDir)
		since, err := syncStore.Load()
		if err != nil && err != store.ErrSyncNotFound {
			app.QueueUpdateDraw(func() { status.SetText(fmt.Sprintf("[red]Error: %v", err)) })
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		dirty, err := local.ListDirty(ctx)
		if err != nil {
			app.QueueUpdateDraw(func() { status.SetText(fmt.Sprintf("[red]Error: %v", err)) })
			return
		}
		if len(dirty) > 0 {
			if _, err := cli.SyncPushEncrypted(ctx, toSyncItems(dirty)); err != nil {
				app.QueueUpdateDraw(func() { status.SetText(fmt.Sprintf("[red]Error: %v", err)) })
				return
			}
			var ids []string
			for _, item := range dirty {
				ids = append(ids, item.ID)
			}
			if err := local.MarkClean(ctx, ids); err != nil {
				app.QueueUpdateDraw(func() { status.SetText(fmt.Sprintf("[red]Error: %v", err)) })
				return
			}
		}

		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		items, err := cli.SyncPullEncrypted(ctx, since)
		if err != nil {
			app.QueueUpdateDraw(func() { status.SetText(fmt.Sprintf("[red]Error: %v", err)) })
			return
		}

		if err := local.ApplyRemote(ctx, fromSyncItems(items)); err != nil {
			app.QueueUpdateDraw(func() { status.SetText(fmt.Sprintf("[red]Error: %v", err)) })
			return
		}

		_ = syncStore.Save(time.Now().UTC())
		app.QueueUpdateDraw(func() { status.SetText(fmt.Sprintf("Synced %d items", len(items))) })
	}()
}

func newLocalID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func toSyncItems(items []store.Item) []api.SyncItem {
	out := make([]api.SyncItem, 0, len(items))
	for _, it := range items {
		out = append(out, api.SyncItem{
			ID:        it.ID,
			Type:      it.Type,
			Payload:   it.Payload,
			Meta:      it.Meta,
			Deleted:   it.Deleted,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}

func fromSyncItems(items []api.SyncItem) []store.Item {
	out := make([]store.Item, 0, len(items))
	for _, it := range items {
		out = append(out, store.Item{
			ID:        it.ID,
			Type:      it.Type,
			Payload:   it.Payload,
			Meta:      it.Meta,
			Deleted:   it.Deleted,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}

func getCrypto(cli *api.Client) *crypto.Crypto {
	return cli.Crypto()
}

func modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}
