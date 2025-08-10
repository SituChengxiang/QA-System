package idgen

import (
	"time"

	"github.com/yitter/idgenerator-go/idgen"
)

// Init 初始化雪花生成器
func Init() {
	var options = idgen.NewIdGeneratorOptions(1)
	options.WorkerIdBitLength = 4
	options.BaseTime = time.Date(2024, 10, 1, 0, 0, 0, 0, time.FixedZone("CST", 8*60*60)).UnixMilli()
	idgen.SetIdGenerator(options)
}
