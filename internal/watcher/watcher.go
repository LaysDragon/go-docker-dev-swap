package watcher

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	path     string
	callback func(string)
}

func New(path string, callback func(string)) *Watcher {
	return &Watcher{
		path:     path,
		callback: callback,
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("建立檔案監控器失敗: %w", err)
	}

	// 監控檔案所在的目錄
	dir := filepath.Dir(w.path)
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("添加監控目錄失敗: %w", err)
	}

	go func() {
		defer watcher.Close()

		// 防抖動計時器
		var debounceTimer *time.Timer
		const debounceDelay = 500 * time.Millisecond

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// 只關注目標檔案的寫入和建立事件
				if event.Name == w.path && (event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create) {

					// 重置防抖動計時器
					if debounceTimer != nil {
						debounceTimer.Stop()
					}

					debounceTimer = time.AfterFunc(debounceDelay, func() {
						w.callback(event.Name)
					})
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("檔案監控錯誤: %v", err)
			}
		}
	}()

	return nil
}
