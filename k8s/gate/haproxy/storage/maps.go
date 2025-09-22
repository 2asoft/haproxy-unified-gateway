// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	futils "github.com/haproxytech/kubernetes-controller/k8s/gate/fileutils"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage/maps"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
)

var _ MapsStorage = &MapsStorageDefault{}

type MapsStorageDefault struct {
	logger     *slog.Logger
	extractGVK utils.ExtractGVK
	mapsDir    string
}

func NewMapsStorage(logger *slog.Logger, extractGVK utils.ExtractGVK, structureType StructureType, mapsDir string) (MapsStorage, error) {
	mylogger := logger.With(logging.LogAttrCategory(logging.LogMapsStorage))

	if mapsDir == "" {
		mapsDir = "/usr/local/hug/maps"
		// return nil, fmt.Errorf("maps directory is not set")
	}

	switch structureType {
	case StructureTypeMapsDefault:
		cs := MapsStorageDefault{
			logger:     mylogger,
			extractGVK: extractGVK,
			mapsDir:    mapsDir,
		}
		cs.empty(mapsDir)
		return &cs, nil
	default:
		return nil, fmt.Errorf("unknown structure type: %s", structureType)
	}
}

func (m *MapsStorageDefault) MapPath(listener string) futils.FilePath {
	// TODO
	_ = listener
	_ = m.mapsDir
	return futils.FilePath{}
}

func (MapsStorageDefault) NewMapData(key, value string) (maps.MapData, error) {
	return maps.MapData{
		Data: map[string]string{
			key: value,
		},
	}, nil
}

func (m *MapsStorageDefault) WriteOnDisk(data maps.MapData) error {
	_ = m.mapsDir
	var f *os.File
	var err error
	if _, err = os.Stat(data.Path.Dir); os.IsNotExist(err) {
		err = os.MkdirAll(data.Path.Dir, 0o755)
		if err != nil {
			return err
		}
	}
	f, err = os.Create(data.Path.FullPath())
	if err != nil {
		return err
	}
	defer f.Close()
	// TODO sort this
	for k, v := range data.Data {
		_, err = f.WriteString(fmt.Sprintf("%s %s\n", k, v))
		if err != nil {
			return err
		}
	}
	return nil
}

func (MapsStorageDefault) DeleteFromDisk(data maps.MapData) error {
	return os.Remove(data.Path.FullPath())
}

func (m *MapsStorageDefault) DeleteEmptyMapsDir() error {
	entries, err := os.ReadDir(m.mapsDir)
	if err != nil {
		return err
	}

	// Iterates over namespace directories
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // We only care about directories
		}

		subdirPath := filepath.Join(m.mapsDir, entry.Name())
		subEntries, err := os.ReadDir(subdirPath)
		if err != nil {
			return err
		}

		if len(subEntries) == 0 {
			m.logger.LogAttrs(context.Background(), slog.LevelDebug, "Deleting empty directory",
				slog.String("dir", subdirPath))
			err := os.RemoveAll(subdirPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *MapsStorageDefault) empty(path string) {
	// TODO
	_ = path
	_ = m.logger
}
