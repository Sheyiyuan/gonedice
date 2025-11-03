# gonedice

这是一个 [OneDice](https://github.com/OlivOS-Team/onedice) 标准的 golang 实现。

## 快速 API

- `func New(expr string, valueTable map[string]int) *RD` —  创建解析器实例；`valueTable` 可用于传入预设变量（键通常为大写）。
- `func (r *RD) Roll()` — 评估表达式并将结果写入内部 `r.res`。
- `func (r *RD) Result() Result` — 返回 `Result` 结果结构。

类型 `Result` 的主要字段：

- `Value int` — 最终整型值（主结果）。
- `Min`, `Max int` — 结果区间（当前实现为简单占位，可在若干运算符上扩展）。
- `Detail string` — 可读的细节摘要（包括 value、meta、temp 与 valueTable 快照）。
- `MetaTuple []interface{}` — 元数据列表，元素可能是 `int`（骰子结果）或 `string`（`lp` 模板等）。
- `Error ErrorType` — 非空表示出错。

`RD` 结构中可直接访问的便利点：

- `r.rng` — 随机数生成器；你可以替换为 `rand.New(rand.NewSource(seed))` 以获得确定性输出（便于测试）。
- `r.ValueTable` — 全局/传入的变量表。

## 使用示例（完整）

```go
package main

import (
	"fmt"
	"math/rand"

	"github.com/Sheyiyuan/onedice"
)

func main() {
	// 简单骰子示例
	r := onedice.New("2d6k1", nil)
	r.rng = rand.New(rand.NewSource(114514)) // 可选：确定性测试
	r.Roll()
	res := r.Result()
	fmt.Println("Value:", res.Value)
	fmt.Println("MetaTuple:", res.MetaTuple)
	fmt.Println("Detail:", res.Detail)

	// lp 字符串模板
	r2 := onedice.New("\"x{i}y\"lp3", nil)
	r2.Roll()
	res2 := r2.Result()
	fmt.Println("LP meta:", res2.MetaTuple)

	// 使用 ValueTable 变量替换
	vt := map[string]int{"STR": 5}
	r3 := onedice.New("{STR}+2", vt)
	r3.Roll()
	fmt.Println("Var result:", r3.Result().Value)
}
```

## 访问 MetaTuple（类型断言）

`MetaTuple` 的元素类型为 `interface{}`，因此你需要用类型断言来区分：

```go
for _, el := range res.MetaTuple {
	switch v := el.(type) {
	case int:
		fmt.Println("int meta:", v)
	case string:
		fmt.Println("string meta:", v)
	default:
		fmt.Println("unknown meta type")
	}
}
```

## 临时变量 `$t` 与 ValueTable 的交互

- 读取 `$t` 时优先使用 `r.temp`；若未设置再查 `r.ValueTable["Tn"]`。
- 赋值操作 `=` 会同时写入 `r.temp` 与 `r.ValueTable["Tn"]`，这样子表达式中读取 `$t` 可以看到父级写入的值。

## 确定性测试（控制 RNG）

在测试或示例中，你可以替换 `r.rng` 以得到可重复的输出：

```go
r.rng = rand.New(rand.NewSource(114514))
```

单元测试中大量使用此手法断言结果。

## 错误处理

在调用 `r.Roll()` 后，请检查 `res.Error` 是否为空。若非空，表示解析或求值阶段出现错误（如语法错误、参数越界、除以零等）。
