package ebschedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getValue(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key": "val",
		}

		var v *string
		found, err := getValue(s, "/key", &v)
		assert.True(found)
		assert.NoError(err)
		assert.Equal("val", *v)
	})

	t.Run("normal-int", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key": 11,
		}

		var v *int
		found, err := getValue(s, "/key", &v)
		assert.True(found)
		assert.NoError(err)
		assert.Equal(11, *v)
	})

	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key": "val",
		}

		var v *string
		found, err := getValue(s, "/notexist", &v)
		assert.False(found)
		assert.NoError(err)
		assert.Nil(v)
	})

	t.Run("type mismatch", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key": "val",
		}

		var v *int
		found, err := getValue(s, "/key", &v)
		assert.True(found)
		assert.EqualError(err, `type mismatch: val=string, dst=*int`)
		assert.Nil(v)
	})
}

func Test_removeValue(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key":  "val",
			"key2": "val2",
		}

		s, removed, err := removeValue(s, "/key")
		assert.True(removed)
		assert.NoError(err)
		assert.Equal(map[string]any{"key2": "val2"}, s)
	})

	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key":  "val",
			"key2": "val2",
		}

		s, removed, err := removeValue(s, "/notexist")
		assert.False(removed)
		assert.NoError(err)
		assert.Equal(map[string]any{
			"key":  "val",
			"key2": "val2",
		}, s)
	})
}

func Test_setValue(t *testing.T) {
	t.Run("override", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key":  "val",
			"key2": "val2",
		}

		err := setValue(s, "/key", "overriden")
		assert.NoError(err)
		assert.Equal(map[string]any{
			"key":  "overriden",
			"key2": "val2",
		}, s)
	})

	t.Run("new", func(t *testing.T) {
		assert := assert.New(t)
		s := map[string]interface{}{
			"key":  "val",
			"key2": "val2",
		}

		err := setValue(s, "/new", "new-value")
		assert.NoError(err)
		assert.Equal(map[string]any{
			"key":  "val",
			"new":  "new-value",
			"key2": "val2",
		}, s)
	})
}
