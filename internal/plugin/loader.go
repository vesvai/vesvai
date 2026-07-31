package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
)

type DiskLoader struct {
	pluginsDir string
}

func NewDiskLoader(pluginsDir string) *DiskLoader {
	return &DiskLoader{
		pluginsDir: pluginsDir,
	}
}

func (l *DiskLoader) Load() ([]Plugin, error) {
	entries, err := os.ReadDir(l.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var plugins []Plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(l.pluginsDir, entry.Name())
		p, err := l.loadPlugin(pluginDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %s: %w", entry.Name(), err)
		}
		if p != nil {
			plugins = append(plugins, p)
		}
	}

	return plugins, nil
}

func (l *DiskLoader) LoadOne(name string) (Plugin, error) {
	pluginDir := filepath.Join(l.pluginsDir, name)
	return l.loadPlugin(pluginDir)
}

func (l *DiskLoader) loadPlugin(dir string) (Plugin, error) {
	metaPath := filepath.Join(dir, "meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read meta.json: %w", err)
	}

	var diskMeta DiskPluginMeta
	if err := json.Unmarshal(metaData, &diskMeta); err != nil {
		return nil, fmt.Errorf("failed to parse meta.json: %w", err)
	}

	if diskMeta.Entry == "" {
		diskMeta.Entry = diskMeta.Name + ".so"
	}

	entryPath := filepath.Join(dir, diskMeta.Entry)
	goPlugin, err := plugin.Open(entryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin %s: %w", entryPath, err)
	}

	symPlugin, err := goPlugin.Lookup("Plugin")
	if err != nil {
		symNewPlugin, err := goPlugin.Lookup("NewPlugin")
		if err != nil {
			return nil, fmt.Errorf("plugin %s does not export Plugin or NewPlugin", dir)
		}

		newPluginFn, ok := symNewPlugin.(func() Plugin)
		if !ok {
			return nil, fmt.Errorf("NewPlugin in %s has wrong signature", dir)
		}

		return newPluginFn(), nil
	}

	p, ok := symPlugin.(Plugin)
	if !ok {
		return nil, fmt.Errorf("Plugin in %s does not implement plugin.Plugin", dir)
	}

	return p, nil
}

func (l *DiskLoader) Discover() ([]DiskPluginMeta, error) {
	entries, err := os.ReadDir(l.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var metas []DiskPluginMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaPath := filepath.Join(l.pluginsDir, entry.Name(), "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var meta DiskPluginMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}

		metas = append(metas, meta)
	}

	return metas, nil
}
