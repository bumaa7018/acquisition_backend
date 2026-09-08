package config

// EXTERNAL_API_KEYS-ийн задлалт. Түлхүүр буруу задарвал гадаад систем
// хандаж чадахгүй (эсвэл бүр дор — хүсээгүй түлхүүр хүчинтэй болно).

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExternalAPIConfig_ParsedKeys(t *testing.T) {
	cfg := ExternalAPIConfig{
		Keys: " teg:key-1 , shshgeg:key-2 ,bare-key, ,:no-label-key, empty-key: ",
	}
	got := cfg.ParsedKeys()

	assert.Len(t, got, 4)
	assert.Equal(t, "teg", got[0].Label)
	assert.Equal(t, "key-1", got[0].Key)
	assert.Equal(t, "shshgeg", got[1].Label)
	assert.Equal(t, "key-2", got[1].Key)
	// Нэргүй бичсэн түлхүүр — нэр хоосон (middleware "external" гэж нэрлэнэ)
	assert.Empty(t, got[2].Label)
	assert.Equal(t, "bare-key", got[2].Key)
	// ":no-label-key" — идэвхтэй ч нэргүй
	assert.Equal(t, "no-label-key", got[3].Key)
}

func TestExternalAPIConfig_EmptyMeansClosed(t *testing.T) {
	assert.Empty(t, ExternalAPIConfig{}.ParsedKeys())
	assert.Empty(t, ExternalAPIConfig{Keys: "  , , "}.ParsedKeys())
}
