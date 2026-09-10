package signal

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchConfig 监听指定配置文件的变化，返回 channel。
// 文件被修改/创建/重命名时会向 channel 发送事件（带 100ms 防抖）。
// 跨平台：底层在 Linux 用 inotify，macOS 用 FSEvents，Windows 用 ReadDirectoryChangesW。
//
// 监听的是配置文件所在的目录（而非文件本身），因为大多数编辑器保存时
// 会采用"原子替换"策略（写临时文件 + rename），直接监听文件会漏掉。
func WatchConfig(path string) (<-chan struct{}, error) {
	out := make(chan struct{}, 1)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, err
	}

	targetName := filepath.Base(path)

	// 防抖：编辑器保存时通常产生多个事件（WRITE + CREATE/RENAME）。
	const debounce = 100 * time.Millisecond
	var timer *time.Timer

	go func() {
		defer close(out)
		defer w.Close()

		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				// 只关心目标文件；并且关心 WRITE/ CREATE/ RENAME/ CHMOD。
				if filepath.Base(ev.Name) != targetName {
					continue
				}
				if !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) && !ev.Has(fsnotify.Rename) && !ev.Has(fsnotify.Chmod) {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounce, func() {
					select {
					case out <- struct{}{}:
					default:
					}
				})
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return out, nil
}