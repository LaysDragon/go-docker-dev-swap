package local

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher 監控本地文件變化
type FileWatcher struct {
	path     string
	callback func(string)
}

// NewFileWatcher 創建文件監控器
func NewFileWatcher(path string, callback func(string)) *FileWatcher {
	return &FileWatcher{
		path:     path,
		callback: callback,
	}
}

// Start 開始監控文件變化
func (fw *FileWatcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("建立檔案監控器失敗: %w", err)
	}

	// 監控檔案所在的目錄
	dir := filepath.Dir(fw.path)
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
				// it event.op always be trigger as chmod event with go build under wsl for some reason
				//if event.Name == fw.path && (event.Op&fsnotify.Write == fsnotify.Write ||
				//	event.Op&fsnotify.Create == fsnotify.Create) {
				if event.Name == fw.path {

					// 重置防抖動計時器
					if debounceTimer != nil {
						debounceTimer.Stop()
					}

					debounceTimer = time.AfterFunc(debounceDelay, func() {
						fw.callback(event.Name)
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
