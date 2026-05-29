package gui

import (
	"fmt"
	"pan/internal/oss"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Browser struct {
	client       *oss.Client
	view         *widget.Table
	items        []oss.ObjectInfo
	prefix       string
	history      []string
	selected     map[int]bool
	onNav        func(string)
	onStatus     func(string)
	onDoubleClick func(oss.ObjectInfo)
	lastClickRow int
	lastClickAt  time.Time
}

func NewBrowser(client *oss.Client, onNav func(string), onStatus func(string), onDbl func(oss.ObjectInfo)) *Browser {
	b := &Browser{
		client:        client,
		prefix:        "",
		selected:      make(map[int]bool),
		onNav:         onNav,
		onStatus:      onStatus,
		onDoubleClick: onDbl,
	}

	b.view = widget.NewTable(
		func() (int, int) { return len(b.items) + 1, 4 },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.SetText("")
			if id.Row == 0 {
				switch id.Col {
				case 0:
					label.SetText("")
				case 1:
					label.SetText("Name")
				case 2:
					label.SetText("Size")
				case 3:
					label.SetText("Modified")
				}
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			idx := id.Row - 1
			if idx >= len(b.items) {
				return
			}
			item := b.items[idx]
			switch id.Col {
			case 0:
				if item.IsDir {
					label.SetText("[DIR]")
				} else {
					label.SetText("[FILE]")
				}
			case 1:
				label.SetText(item.Name)
			case 2:
				if item.IsDir {
					label.SetText("-")
				} else {
					label.SetText(humanSize(item.Size))
				}
			case 3:
				if item.IsDir {
					label.SetText("-")
				} else if !item.LastModified.IsZero() {
					label.SetText(item.LastModified.Format("2006-01-02 15:04"))
				}
			}
		},
	)

	b.view.SetColumnWidth(0, 60)
	b.view.SetColumnWidth(1, 300)
	b.view.SetColumnWidth(2, 100)
	b.view.SetColumnWidth(3, 160)

	b.view.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			return
		}
		idx := id.Row - 1
		if idx >= len(b.items) {
			return
		}

		now := time.Now()
		if b.lastClickRow == id.Row && now.Sub(b.lastClickAt) < 400*time.Millisecond {
			b.handleDoubleClick(idx)
			b.lastClickRow = 0
			return
		}
		b.lastClickRow = id.Row
		b.lastClickAt = now

		if b.selected[idx] {
			delete(b.selected, idx)
		} else {
			b.selected[idx] = true
		}
		b.updateStatus()
		b.view.Refresh()
	}

	return b
}

func (b *Browser) Load(prefix string) error {
	items, err := b.client.ListDir(prefix)
	if err != nil {
		return fmt.Errorf("load %q: %w", prefix, err)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})

	b.prefix = prefix
	b.items = items
	b.selected = make(map[int]bool)
	b.onNav(b.formatPath(prefix))
	b.view.Refresh()
	b.updateStatus()
	return nil
}

func (b *Browser) NavigateTo(name string) {
	if b.prefix != "" || name != ".." {
		b.history = append(b.history, b.prefix)
	}
	var newPrefix string
	if name == ".." {
		newPrefix = b.parentPrefix(b.prefix)
	} else {
		if b.prefix == "" {
			newPrefix = name + "/"
		} else {
			newPrefix = b.prefix + name + "/"
		}
	}
	b.Load(newPrefix)
}

func (b *Browser) NavigateToFile(item oss.ObjectInfo) {
	if item.IsDir {
		b.NavigateTo(item.Name)
	}
}

func (b *Browser) GoBack() {
	if len(b.history) == 0 {
		if b.prefix != "" {
			b.Load("")
		}
		return
	}
	prev := b.history[len(b.history)-1]
	b.history = b.history[:len(b.history)-1]
	b.Load(prev)
}

func (b *Browser) handleDoubleClick(idx int) {
	if idx >= len(b.items) {
		return
	}
	item := b.items[idx]
	b.selected = make(map[int]bool)
	b.selected[idx] = true
	b.updateStatus()
	b.view.Refresh()

	if b.onDoubleClick != nil {
		b.onDoubleClick(item)
	}
}

func (b *Browser) DownloadSelected(win fyne.Window) {
	for idx := range b.selected {
		if idx >= len(b.items) {
			continue
		}
		item := b.items[idx]
		if item.IsDir {
			continue
		}
		b.downloadItem(item, win)
	}
}

func (b *Browser) downloadItem(item oss.ObjectInfo, win fyne.Window) {
	progress := widget.NewProgressBar()
	progressDlg := dialog.NewCustom("Downloading", "Cancel", progress, win)
	progressDlg.Show()

	localPath := item.Name

	go func() {
		err := b.client.DownloadFileWithProgress(item.Key, localPath, func(downloaded, total int64) {
			if total > 0 {
				progress.SetValue(float64(downloaded) / float64(total))
			}
		})
		progressDlg.Hide()
		if err != nil {
			dialog.ShowError(fmt.Errorf("download %s: %w", item.Name, err), win)
		}
	}()
}

func (b *Browser) SelectedItems() []oss.ObjectInfo {
	var result []oss.ObjectInfo
	for idx := range b.selected {
		if idx < len(b.items) {
			result = append(result, b.items[idx])
		}
	}
	return result
}

func (b *Browser) Prefix() string {
	return b.prefix
}

func (b *Browser) updateStatus() {
	count := len(b.items)
	sel := len(b.selected)
	b.onStatus(fmt.Sprintf("%s, %s selected", humanCount(count, "item"), humanCount(sel, "item")))
}

func (b *Browser) formatPath(prefix string) string {
	if prefix == "" {
		return "/"
	}
	return "/" + prefix
}

func (b *Browser) parentPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	s := prefix
	if s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return s[:i+1]
		}
	}
	return ""
}
