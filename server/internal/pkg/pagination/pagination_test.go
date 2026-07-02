package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type item struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:32"`
}

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&item{}))
	for i := 1; i <= 25; i++ {
		db.Create(&item{Name: "n"})
	}
	return db
}

func TestQuery_Normalize(t *testing.T) {
	q := Query{Page: 0, Size: 0}
	q.Normalize()
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 10, q.Size)

	q2 := Query{Page: -1, Size: 200}
	q2.Normalize()
	assert.Equal(t, 1, q2.Page)
	assert.Equal(t, 100, q2.Size) // 上限 100
}

func TestPaginate(t *testing.T) {
	db := setupDB(t)
	q := Query{Page: 2, Size: 10}
	q.Normalize()
	res, err := Paginate[item](db, q, func(d *gorm.DB) *gorm.DB { return d })
	require.NoError(t, err)
	assert.Equal(t, int64(25), res.Total)
	assert.Equal(t, 2, res.Current)
	assert.Equal(t, 10, res.Size)
	assert.Len(t, res.Records, 10) // 第二页 10 条
}

func TestPaginate_Keyword(t *testing.T) {
	db := setupDB(t)
	// 建 5 条带特殊名字
	for i := 0; i < 5; i++ {
		db.Create(&item{Name: "special"})
	}
	q := Query{Page: 1, Size: 10, Keyword: "special"}
	q.Normalize()
	res, err := Paginate[item](db, q, func(d *gorm.DB) *gorm.DB {
		return d.Where("name = ?", "special") // build 回调体现 keyword
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), res.Total)
	assert.Len(t, res.Records, 5)
}
