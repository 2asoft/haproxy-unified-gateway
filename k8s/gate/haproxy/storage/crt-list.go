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
	"sort"
	"strings"

	futils "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/fileutils"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/certificate"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CrtListStorage interface {
	// CertListPath returns the FilePath for the crt-file file.
	CertListPath(listenerKey client.ObjectKey) futils.FilePath
	// DeleteCrtListFromDisk deletes a crt-list from disk
	DeleteCrtListFromDisk(crtListData certificate.CrtListData) error
	// NewCrtListData returns the new CrtListData
	NewCrtListData(listenerKey client.ObjectKey, secretKeys map[client.ObjectKey]struct{}) certificate.CrtListData
	// WriteCrtListOnDisk writes a new crt-list on disk
	WriteCrtListOnDisk(crtList certificate.CrtListData) error
	// UpdateCrtListOnDisk updates a crt-list on disk with new certificates and removed ones
	UpdateCrtListOnDisk(gatewayKey client.ObjectKey, newSecretKeys, removedSecretKeys map[client.ObjectKey]struct{}) error
}

var _ CrtListStorage = &CertificateStorageDefault{}

func (c *CertificateStorageDefault) CertListPath(listenerKey client.ObjectKey) futils.FilePath {
	return futils.FilePath{
		Dir:      c.CertFilesBaseDir,
		FileName: fmt.Sprintf("%s_%s.list", listenerKey.Namespace, listenerKey.Name),
	}
}

func (c *CertificateStorageDefault) NewCrtListData(listenerKey client.ObjectKey, secretKeys map[client.ObjectKey]struct{}) certificate.CrtListData {
	certFullPaths := make([]string, 0, len(secretKeys))
	for secretKey := range secretKeys {
		certFullPaths = append(certFullPaths, c.CertPath(secretKey).FullPath())
	}
	sort.Strings(certFullPaths)
	crtListPath := c.CertListPath(listenerKey)
	return certificate.NewCrtListData(crtListPath, certFullPaths)
}

func (c *CertificateStorageDefault) WriteCrtListOnDisk(crtList certificate.CrtListData) error {
	crtListFullPath := crtList.Path.FullPath()

	// SORT them
	sort.Strings(crtList.Content)
	// Now write sorted paths
	var builder strings.Builder
	for _, crtFullPath := range crtList.Content {
		builder.WriteString(crtFullPath)
		builder.WriteString("\n")
	}

	// Create or open the file for writing.
	file, err := os.Create(crtListFullPath)
	if err != nil {
		c.logger.LogAttrs(context.Background(), slog.LevelError, "crt-list [not written]",
			slog.String("crt-list", crtListFullPath),
		)
		return err
	}
	// Use a defer statement to ensure the file is closed at the end of the function.
	defer file.Close()

	// Write the entire string from the builder to the file in a single operation.
	_, err = file.WriteString(builder.String())
	if err != nil {
		c.logger.LogAttrs(context.Background(), slog.LevelError, "crt-list [not written]",
			slog.String("crt-list", crtListFullPath),
		)
		return err
	}

	c.logger.LogAttrs(context.Background(), slog.LevelInfo, "crt-list [written]",
		slog.String("crt-list", crtListFullPath),
	)
	return nil
}

func (c *CertificateStorageDefault) DeleteCrtListFromDisk(crtListData certificate.CrtListData) error {
	crtListFilePath := crtListData.Path
	fullPath := crtListFilePath.FullPath()

	err := crtListFilePath.DeleteFromDisk()
	if err == nil {
		c.logger.LogAttrs(context.Background(), slog.LevelInfo, "crt-list [deleted]",
			slog.String("crt-list", fullPath))
		return nil
	}
	if os.IsNotExist(err) {
		return nil
	}

	c.logger.LogAttrs(context.Background(), slog.LevelError, "crt-list [not deleted] from disk",
		slog.String("crt-list", fullPath),
		logging.LogAttrError(err))
	return err
}

func (c *CertificateStorageDefault) UpdateCrtListOnDisk(listenerKey client.ObjectKey, newSecretKeys, removedSecretKeys map[client.ObjectKey]struct{}) error {
	// No change, nothing to do
	if len(newSecretKeys) == 0 && len(removedSecretKeys) == 0 {
		return nil
	}

	crtListFilePath := c.CertListPath(listenerKey)
	crtlistFullPath := crtListFilePath.FullPath()

	crtFullPaths, err := crtListFilePath.ReadLines()
	if err != nil {
		c.logger.LogAttrs(context.Background(), slog.LevelError, "crt-list [not written]",
			slog.String("crt-list", crtlistFullPath),
			logging.LogAttrKey(listenerKey), logging.LogAttrError(err))
		return err
	}

	secretKeys := make(map[client.ObjectKey]struct{}, len(crtFullPaths))

	for _, crtFullPath := range crtFullPaths {
		secretKey, err := c.secretKeyFromCertificateFullPath(crtFullPath)
		if err != nil {
			c.logger.LogAttrs(context.Background(), slog.LevelError, "crt-list [not written]",
				slog.String("crt-list", crtlistFullPath),
				logging.LogAttrKey(listenerKey), logging.LogAttrError(err))
			return err
		}
		secretKeys[secretKey] = struct{}{}
	}
	// Append the new ones
	for secretKey := range newSecretKeys {
		secretKeys[secretKey] = struct{}{}
	}

	// Remove the dereferenced ones
	for secretKey := range removedSecretKeys {
		delete(secretKeys, secretKey)
	}

	// and re-write on disk
	// return c.WriteCrtListOnDisk(gatewayKey, secretKeys)
	return nil
}

func (*CertificateStorageDefault) secretKeyFromCertificateFullPath(path string) (client.ObjectKey, error) {
	withoutExt := removeFileExtension(path)
	fileName := filepath.Base(withoutExt)
	parts := strings.Split(fileName, "_")
	if len(parts) != 2 {
		return types.NamespacedName{}, fmt.Errorf("invalid cert file name: %s", path)
	}
	return client.ObjectKey{Namespace: parts[0], Name: parts[1]}, nil
}

// removeFileExtension takes a filename as a string and removes its extension.
// For example, "document.txt" becomes "document".
func removeFileExtension(filename string) string {
	// Get the file extension, including the dot.
	ext := filepath.Ext(filename)
	// Return the string with the extension removed.
	return strings.TrimSuffix(filename, ext)
}
