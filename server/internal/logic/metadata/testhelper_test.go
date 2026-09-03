package metadata

import (
	"context"
	"testing"
)

// newTestDB creates an in-memory database ready for testing.
func newTestDB(t *testing.T) *Database {
	t.Helper()
	db, err := NewDatabase(context.Background(), DatabaseOptions{
		StoragePath: ":memory:",
		TablePrefix: "t_",
	})
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	if err := db.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	t.Cleanup(func() {
		db.Close(context.Background())
	})
	return db
}

// createBasicCollection creates a collection with standard fields for testing.
func createBasicCollection(t *testing.T, db *Database, name string) *Collection {
	t.Helper()
	statusEnum := []interface{}{"draft", "published", "archived"}
	coll, err := db.CreateCollection(context.Background(), CreateCollectionInput{
		Name:         name,
		DisplayName:  name + "_display",
		PresetFields: []string{"id", "createdAt", "updatedAt"},
		Fields: []CreateFieldInput{
			{Name: "title", Type: "string", DisplayName: "标题", IsRequired: true, Length: 200},
			{Name: "age", Type: "integer", DisplayName: "年龄"},
			{Name: "status", Type: "select", DisplayName: "状态", Options: map[string]interface{}{
				"enum": statusEnum,
			}},
		},
	})
	if err != nil {
		t.Fatalf("createBasicCollection(%s): %v", name, err)
	}
	return coll
}

// seedData seeds n records into the given repo. Returns the created records.
func seedData(t *testing.T, repo Repository, n int) []*Record {
	t.Helper()
	ctx := context.Background()
	records := make([]*Record, 0, n)
	for i := 0; i < n; i++ {
		status := "draft"
		if i%2 == 0 {
			status = "published"
		}
		r, err := repo.Create(ctx, &CreateOptions{
			Values: map[string]interface{}{
				"title":  "item_" + itoa(i),
				"age":    18 + i,
				"status": status,
			},
		})
		if err != nil {
			t.Fatalf("seedData create: %v", err)
		}
		records = append(records, r)
	}
	return records
}

// itoa is a simple int->string helper to avoid importing strconv everywhere.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
