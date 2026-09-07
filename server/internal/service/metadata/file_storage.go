package metadata

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"itcodex/server/internal/consts"
	"itcodex/server/pkg/utils"
)

func (d *Database) StoragePath() string {
	if d == nil {
		return consts.DefaultStoragePath
	}
	if p := strings.TrimSpace(d.options.StoragePath); p != "" {
		return p
	}
	return consts.DefaultStoragePath
}

func (d *Database) SaveFileObject(ctx context.Context, collection, originalName, contentType string, r io.Reader) (map[string]any, error) {
	coll := d.Collection(collection)
	if coll == nil {
		return nil, NewNotFoundError("集合", "name", collection)
	}
	if coll.Type() != CollectionTypeFile {
		return nil, NewForbiddenError("仅文件表支持上传")
	}

	safeName := filepath.Base(originalName)
	if safeName == "" || safeName == "." || safeName == string(filepath.Separator) {
		safeName = "file"
	}
	id := utils.NextID()
	rel := filepath.ToSlash(filepath.Join(collection, fmt.Sprintf("%d_%s", id, safeName)))
	abs := filepath.Join(d.StoragePath(), rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, NewSystemError(err)
	}
	f, err := os.Create(abs)
	if err != nil {
		return nil, NewSystemError(err)
	}
	n, err := io.Copy(f, r)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(abs)
		return nil, NewSystemError(err)
	}
	if contentType == "" || contentType == "application/octet-stream" {
		if guessed := mime.TypeByExtension(filepath.Ext(safeName)); guessed != "" {
			contentType = guessed
		}
	}
	url := "http://local/api/c/" + collection + "/files/" + fmt.Sprintf("%d", id) + "/content"
	rec, err := coll.Repository().Create(ctx, &CreateOptions{Values: map[string]any{
		"id":   id,
		"name": safeName,
		"url":  url,
		"mime": contentType,
		"size": n,
		"path": rel,
	}})
	if err != nil {
		_ = os.Remove(abs)
		return nil, err
	}
	return rec.Data(), nil
}

func (d *Database) OpenFileObject(ctx context.Context, collection string, id any) (absPath, contentType, downloadName string, err error) {
	coll := d.Collection(collection)
	if coll == nil {
		return "", "", "", NewNotFoundError("集合", "name", collection)
	}
	rec, err := coll.Repository().FindOne(ctx, &FindOneOptions{FilterByTk: id})
	if err != nil {
		return "", "", "", err
	}
	rel := strings.TrimSpace(castToString(rec.Get("path")))
	if rel == "" {
		rel = strings.TrimPrefix(castToString(rec.Get("url")), "/files/")
	}
	if rel == "" {
		return "", "", "", NewNotFoundError("文件", "id", id)
	}
	abs := filepath.Join(d.StoragePath(), filepath.FromSlash(rel))
	if _, statErr := os.Stat(abs); statErr != nil {
		return "", "", "", NewNotFoundError("文件", "path", rel)
	}
	return abs, castToString(rec.Get("mime")), castToString(rec.Get("name")), nil
}
