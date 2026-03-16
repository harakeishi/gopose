package types

// Field はログフィールドのキー・値ペアを表します。
type Field struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
