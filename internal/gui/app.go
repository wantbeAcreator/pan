package gui

import (
	"fmt"
	"os"
	"pan/internal/oss"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type App struct {
	fyneApp  fyne.App
	window   fyne.Window
	client   *oss.Client
	browser  *Browser
	addrBar  *widget.Label
	status   *widget.Label
}

func Start() {
	fmt.Fprintln(os.Stderr, "gui: creating app...")
	guiApp := &App{
		fyneApp: app.NewWithID("pan"),
	}

	guiApp.fyneApp.Settings().SetTheme(theme.LightTheme())

	fmt.Fprintln(os.Stderr, "gui: creating window...")
	guiApp.window = guiApp.fyneApp.NewWindow("Pan - OSS File Manager")
	guiApp.window.Resize(fyne.NewSize(900, 600))

	guiApp.addrBar = widget.NewLabel("/")
	guiApp.status = widget.NewLabel("Connecting...")

	fmt.Fprintln(os.Stderr, "gui: connecting OSS...")
	client, err := oss.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gui: OSS error: %v\n", err)
		dialog.ShowError(err, guiApp.window)
		guiApp.status.SetText("Connection failed")
		guiApp.window.ShowAndRun()
		return
	}
	guiApp.client = client

	fmt.Fprintln(os.Stderr, "gui: building browser...")
	guiApp.browser = NewBrowser(client, guiApp.onNavigate, guiApp.onStatus, guiApp.onDoubleClick)

	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		guiApp.browser.GoBack()
	})
	uploadBtn := widget.NewButtonWithIcon("Upload", theme.UploadIcon(), func() {
		guiApp.showUploadDialog()
	})
	downloadBtn := widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
		guiApp.browser.DownloadSelected(guiApp.window)
	})
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		guiApp.showDeleteConfirm()
	})
	newDirBtn := widget.NewButtonWithIcon("New Folder", theme.FolderNewIcon(), func() {
		guiApp.showNewDirDialog()
	})
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		guiApp.refresh()
	})

	toolbar := container.NewHBox(
		backBtn, uploadBtn, downloadBtn, deleteBtn, newDirBtn, refreshBtn,
	)

	content := container.NewBorder(
		container.NewVBox(toolbar, guiApp.addrBar),
		container.NewHBox(guiApp.status),
		nil, nil,
		guiApp.browser.view,
	)

	guiApp.window.SetContent(content)
	guiApp.refresh()

	fmt.Fprintln(os.Stderr, "gui: starting main loop...")
	guiApp.window.ShowAndRun()
}

func (a *App) onNavigate(prefix string) {
	a.addrBar.SetText(prefix)
}

func (a *App) onStatus(msg string) {
	a.status.SetText(msg)
}

func (a *App) onDoubleClick(item oss.ObjectInfo) {
	if item.IsDir {
		a.browser.NavigateTo(item.Name)
	} else {
		a.browser.downloadItem(item, a.window)
	}
}

func (a *App) refresh() {
	if err := a.browser.Load(""); err != nil {
		dialog.ShowError(err, a.window)
	}
}

func (a *App) showUploadDialog() {
	dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		progress := widget.NewProgressBar()
		progressDlg := dialog.NewCustom("Uploading", "Cancel", progress, a.window)
		progressDlg.Show()

		localPath := reader.URI().Path()
		remoteKey := a.browser.Prefix() + reader.URI().Name()

		go func() {
			err := a.client.UploadWithProgress(localPath, remoteKey, func(uploaded, total int64) {
				if total > 0 {
					progress.SetValue(float64(uploaded) / float64(total))
				}
			})
			progressDlg.Hide()
			if err != nil {
				dialog.ShowError(err, a.window)
			} else {
				a.refresh()
			}
		}()
	}, a.window)
	dlg.Show()
}

func (a *App) showDeleteConfirm() {
	selected := a.browser.SelectedItems()
	if len(selected) == 0 {
		dialog.ShowInformation("Delete", "No items selected", a.window)
		return
	}

	msg := "Delete " + humanCount(len(selected), "item") + "?"
	dlg := dialog.NewConfirm("Delete", msg, func(ok bool) {
		if !ok {
			return
		}
		var keys []string
		for _, item := range selected {
			keys = append(keys, item.Key)
		}
		if err := a.client.DeleteBatch(keys); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		a.refresh()
	}, a.window)
	dlg.Show()
}

func (a *App) showNewDirDialog() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("folder name")
	dlg := dialog.NewCustomConfirm("New Folder", "Create", "Cancel", entry, func(ok bool) {
		if !ok || entry.Text == "" {
			return
		}
		key := a.browser.Prefix() + entry.Text + "/"
		if err := a.client.PutDir(key); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		a.refresh()
	}, a.window)
	dlg.Show()
}
