package infra

import (
	"errors"
	"io/fs"
	"sync"
)

var (
	errAssetFSMissing = errors.New("asset fs not set")

	asset struct {
		mutex sync.RWMutex
		fsys  fs.FS
	}
)

// AssetFS sets/gets global asset filesystem.
// call AssetFS(fsys) to set; call AssetFS() to get.
func AssetFS(fss ...fs.FS) fs.FS {
	if len(fss) > 0 {
		asset.mutex.Lock()
		asset.fsys = fss[0]
		asset.mutex.Unlock()
	}

	asset.mutex.RLock()
	defer asset.mutex.RUnlock()
	return asset.fsys
}

// AssetDir reads one directory from global asset filesystem.
func AssetDir(name string) ([]fs.DirEntry, error) {
	fsys := AssetFS()
	if fsys == nil {
		return nil, errAssetFSMissing
	}
	return fs.ReadDir(fsys, name)
}

// AssetFile reads one file from global asset filesystem.
func AssetFile(name string) ([]byte, error) {
	fsys := AssetFS()
	if fsys == nil {
		return nil, errAssetFSMissing
	}
	return fs.ReadFile(fsys, name)
}

// AssetStat stats one file from global asset filesystem.
func AssetStat(name string) (fs.FileInfo, error) {
	fsys := AssetFS()
	if fsys == nil {
		return nil, errAssetFSMissing
	}
	return fs.Stat(fsys, name)
}
