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

	// import the module as defined in go.mod
	"github.com/Sheyiyuan/gonedice"
)

func main() {
	// 简单骰子示例
	r := gonedice.New("2d6k1", nil)
	r.rng = rand.New(rand.NewSource(114514)) // 可选：确定性测试
	r.Roll()
	res := r.Result()
	fmt.Println("Value:", res.Value)
	fmt.Println("MetaTuple:", res.MetaTuple)
	fmt.Println("Detail:", res.Detail)

	// lp 字符串模板
	r2 := gonedice.New("\"x{i}y\"lp3", nil)
	r2.Roll()
	res2 := r2.Result()
	fmt.Println("LP meta:", res2.MetaTuple)

	// 使用 ValueTable 变量替换
	vt := map[string]int{"STR": 5}
	r3 := gonedice.New("{STR}+2", vt)
	r3.Roll()
	fmt.Println("Var result:", r3.Result().Value)
}
```

## 命令行交互 (CLI)

本仓库提供一个简单的交互式命令行入口，可以即时输入 OneDice 表达式并得到结果或错误提示。

构建并运行（在项目根目录）：

```bash
# 以 go run 方式运行（推荐在项目根目录）
go run ./cmd/gonedice

# 或先安装为可执行程序，然后直接调用（模块路径见 go.mod）
go install github.com/Sheyiyuan/gonedice/cmd/gonedice@latest
gonedice
```

交互示例：

```
> 2d6k1
Value: 5
Meta: [3 2]
Detail: 5 [3,2]
> 3a5m6
Value: 2
Meta: [6 1]
> quit
```

CLI 说明：
- 直接输入表达式并回车会输出 Value、Meta（MetaTuple）、以及 Detail（可读摘要）。
- 输入 `quit` 或 `exit` 退出。

## 多元组（`[]`）与多态示例

OneDice 规范中多元组（用 `[...]` 包裹并以逗号分隔）既可以作为值序列，也可以根据上下文“多态”地参与后续运算。

- 逗号表达式（默认取值）：对于不与多元组结合的运算符，多元组的值通常等于最后一个元素的值。例如：

	- `[2,3]d100` 实际上等同于 `3d100`（多元组的最后一项 `3` 作为 `d` 的左值）。

- 与 `kh`/`kl` 等需要多元组的运算符结合：这些运算符会把左侧表达式的多元组结果展开并按需选取/裁切。例如：

	- `[4,2,6]kh2` 会对多元组按降序 `[6,4,2]` 排序并取左侧两个元素，结果的 `MetaTuple` 为 `[6,4]`，Value 为 `10`。
	- 同样地，`[4,2,6]kl2` 会取最小的两个，`MetaTuple` 为 `[2,4]`，Value 为 `6`。

- 元素可以是子表达式：多元组中的元素可以是复杂表达式（例如 `d`、`f`、字符串模板等）；在需要对元素逐一比较或裁切时，库会先评估这些子表达式并使用它们的值作为比较依据。例如：

	- `[1d1,2]kh1` 会先评估 `1d1`（恒等于 1），然后与 `2` 比大小，最终 `kh1` 选出 `2`。

示例（Go 代码片段）：

```go
r := gonedice.New("[4,2,6]kh2", nil)
r.Roll()
res := r.Result()
fmt.Println(res.Value)      // 10
fmt.Println(res.MetaTuple)  // [6 4]

r2 := gonedice.New("[2,3]d6", nil)
r2.rng = rand.New(rand.NewSource(42))
r2.Roll()
// 等同于 New("3d6", nil) 在求值上（多元组默认取最后一项作为标量）
fmt.Println(r2.Result().Value)
```

注意事项：

- 当多元组与仅接受标量的运算符结合（如 `d`）时，通常使用多元组的“最后一个”元素作为标量值（这与 OneDice 的多态规则一致）。
- 当多元组的元素是字符串模板（`lp` 的情形）或无法解析为数值的表达式时，某些需要数值列表的运算符会先尝试求值每个元素；若无法解析，运算会返回错误。

如果你希望我把这些示例在 README 中做成更完整的交互式会话（带确定性 seed 的输出），我可以再补充一小节示范。 


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
