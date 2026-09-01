package vfs

import (
	"github.com/vesvai/vesvai/internal/core/hook"
)

type TransformContext struct {
	Path    string
	Content string
}

type FileDelete struct {
	Path string
}

type vfsHooks struct {
	beforeRead *hook.Hook[TransformContext]
	afterWrite *hook.Hook[TransformContext]
	onDelete   *hook.Hook[FileDelete]
}

func newHooks() vfsHooks {
	return vfsHooks{
		beforeRead: hook.NewHook[TransformContext](),
		afterWrite: hook.NewHook[TransformContext](),
		onDelete:   hook.NewHook[FileDelete](),
	}
}

func (v *VFS) OnBeforeRead(fn func(path, content string) string) {
	v.hooks.beforeRead.Add(func(tc TransformContext) TransformContext {
		tc.Content = fn(tc.Path, tc.Content)
		return tc
	})
}

func (v *VFS) OnAfterWrite(fn func(path, content string) string) {
	v.hooks.afterWrite.Add(func(tc TransformContext) TransformContext {
		tc.Content = fn(tc.Path, tc.Content)
		return tc
	})
}

func (v *VFS) OnFileDelete(fn func(FileDelete) FileDelete) {
	v.hooks.onDelete.Add(fn)
}
