package gonedice

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrorType 表示可能发生的错误类型
type ErrorType string

const (
	// ErrUnknownGenerate 表示未知的生成错误
	ErrUnknownGenerate ErrorType = "UNKNOWN_GENERATE_FATAL 未知的生成错误"
	// ErrInputRawInvalid 表示输入表达式无效
	ErrInputRawInvalid ErrorType = "INPUT_RAW_INVALID 输入表达式无效"
	// ErrNodeStackEmpty 表示节点栈为空
	ErrNodeStackEmpty ErrorType = "NODE_STACK_EMPTY 节点栈为空"
	// ErrNodeLeftValInvalid 表示节点左侧值无效
	ErrNodeLeftValInvalid ErrorType = "NODE_LEFT_VAL_INVALID 节点左侧值无效"
	// ErrNodeRightValInvalid 表示节点右侧值无效
	ErrNodeRightValInvalid ErrorType = "NODE_RIGHT_VAL_INVALID 节点右侧值无效"
)

// Result 保存一次掷骰的结果
type Result struct {
	// Value 最终计算结果
	Value int
	// Min 可能的最小值
	Min int
	// Max 可能的最大值
	Max int
	// Detail 详细的结果描述
	Detail string
	// MetaTuple 元数据列表，包含骰子的具体结果
	MetaTuple []interface{}
	// Error 错误类型，如果没有错误则为空
	Error ErrorType
}

// RD 是掷骰表达式执行器
type RD struct {
	// Expr 原始表达式
	Expr string
	// origin 转换为小写的表达式
	origin string
	// ValueTable 变量值表，用于替换表达式中的变量
	ValueTable map[string]int
	// rng 随机数生成器
	rng *rand.Rand
	// res 计算结果
	res Result
	// temp 临时变量表
	temp map[int]int
	// DefaultFaces 默认骰子面数
	DefaultFaces int
}

// New 创建一个新的 RD 实例
// expr 是要计算的掷骰表达式
// valueTable 是变量值映射表，用于替换表达式中的变量
func New(expr string, valueTable map[string]int) *RD {
	src := expr
	return &RD{
		Expr:         expr,
		origin:       strings.ToLower(src),
		ValueTable:   valueTable,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		temp:         map[int]int{},
		DefaultFaces: 100,
	}
}

// Roll 评估表达式并填充 Result
// 支持数字、四则运算、括号、变量替换 {VAR} 以及基本的 d (NdM) 掷骰
func (r *RD) Roll() {
	expr, err := r.replaceVars(r.origin)
	if err != nil {
		r.res.Error = ErrInputRawInvalid
		return
	}

	tokens, terr := tokenize(expr)
	if terr != nil {
		r.res.Error = ErrInputRawInvalid
		return
	}

	val, derr := r.evalTokens(tokens)
	if derr != "" {
		r.res.Error = derr
		return
	}

	r.res.Value = val.V
	r.res.Min = val.V
	r.res.Max = val.V

	r.res.Detail = r.buildDetail(val)

	if val.MetaEnable {
		if val.MetaStr != nil && len(val.MetaStr) > 0 {
			meta := make([]interface{}, len(val.MetaStr))
			for i, vv := range val.MetaStr {
				meta[i] = vv
			}
			r.res.MetaTuple = meta
		} else {
			meta := make([]interface{}, len(val.Meta))
			for i, vv := range val.Meta {
				meta[i] = vv
			}
			r.res.MetaTuple = meta

			resolved := r.getFromMetaTuple(meta, false, true)
			if len(resolved) == len(meta) {
				meta2 := make([]interface{}, len(resolved))
				for i, v := range resolved {
					meta2[i] = v
				}
				r.res.MetaTuple = meta2
			}
		}
	}

	r.res.Error = ""
}

// buildDetail 构建可读的结果描述
// 包含值、元数据列表以及可选的临时变量与ValueTable快照用于调试
func (r *RD) buildDetail(val Value) string {
	parts := []string{}
	parts = append(parts, fmt.Sprintf("%d", val.V))

	if val.MetaEnable {
		if val.MetaStr != nil && len(val.MetaStr) > 0 {
			items := make([]string, 0, len(val.MetaStr))
			for _, s := range val.MetaStr {
				items = append(items, fmt.Sprintf("\"%s\"", s))
			}
			parts = append(parts, fmt.Sprintf("[%s]", strings.Join(items, ",")))
		} else if val.Meta != nil {
			items := make([]string, 0, len(val.Meta))
			for _, v := range val.Meta {
				items = append(items, strconv.Itoa(v))
			}
			parts = append(parts, fmt.Sprintf("[%s]", strings.Join(items, ",")))
		}
	}

	if val.V != 0 {
		if r.res.Min != r.res.Max {
			parts = append(parts, fmt.Sprintf("min=%d", r.res.Min))
			parts = append(parts, fmt.Sprintf("max=%d", r.res.Max))
		}
	}

	if r.temp != nil && len(r.temp) > 0 {
		keys := make([]int, 0, len(r.temp))
		for k := range r.temp {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		kvs := make([]string, 0, len(keys))
		for _, k := range keys {
			kvs = append(kvs, fmt.Sprintf("t%d=%d", k, r.temp[k]))
		}
		parts = append(parts, fmt.Sprintf("temp:{%s}", strings.Join(kvs, ",")))
	}

	if r.ValueTable != nil && len(r.ValueTable) > 0 {
		keys := make([]string, 0, len(r.ValueTable))
		for k := range r.ValueTable {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		kvs := make([]string, 0, len(keys))
		for _, k := range keys {
			kvs = append(kvs, fmt.Sprintf("%s=%d", k, r.ValueTable[k]))
		}
		parts = append(parts, fmt.Sprintf("vt:{%s}", strings.Join(kvs, ",")))
	}

	return strings.Join(parts, " ")
}

// Result 返回计算结果
func (r *RD) Result() Result {
	return r.res
}

// getFromMetaTuple 评估可能包含整数或字符串表达式的元数据元素切片
// flagLast: 如果为true，将最后一个子结果字段传播到调用者的Result
// flagUpdate: 如果为true，在可用时将子RD的ValueTable合并到调用者的ValueTable
// 返回成功评估的元素的整数切片，失败时返回空切片
func (r *RD) getFromMetaTuple(data []interface{}, flagLast bool, flagUpdate bool) []int {
	res := make([]int, 0, len(data))
	for _, el := range data {
		switch v := el.(type) {
		case int:
			res = append(res, v)
		case string:
			var subVT map[string]int
			if flagUpdate {
				subVT = r.ValueTable
			} else if r.ValueTable != nil {
				subVT = make(map[string]int, len(r.ValueTable))
				for kk, vv := range r.ValueTable {
					subVT[kk] = vv
				}
			} else {
				subVT = nil
			}

			sub := New(v, subVT)
			sub.rng = r.rng
			sub.Roll()

			if sub.res.Error == "" {
				res = append(res, sub.res.Value)

				if flagUpdate && sub.ValueTable != nil {
					if r.ValueTable == nil {
						r.ValueTable = map[string]int{}
					}
					for kk, vv := range sub.ValueTable {
						r.ValueTable[kk] = vv
					}
				}

				if flagLast {
					r.res.Value = sub.res.Value
					r.res.Min = sub.res.Min
					r.res.Max = sub.res.Max
					r.res.Detail = sub.res.Detail
					r.res.Error = sub.res.Error
				}
			} else {
				return []int{}
			}
		default:
			return []int{}
		}
	}
	return res
}

var varRe = regexp.MustCompile(`\{([^}]+)\}`)

// replaceVars 替换表达式中的变量
func (r *RD) replaceVars(s string) (string, error) {
	if r.ValueTable == nil {
		return s, nil
	}

	out := varRe.ReplaceAllStringFunc(s, func(m string) string {
		key := strings.Trim(m, "{}")
		up := strings.ToUpper(key)
		if v, ok := r.ValueTable[up]; ok {
			return strconv.Itoa(v)
		}
		return m
	})

	return out, nil
}

// isDigit 判断字符是否为数字
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// Value 保存运行时值和可选的骰子结果元数据
type Value struct {
	// V 数值
	V int
	// Meta 整数类型的元数据
	Meta []int
	// MetaEnable 是否启用元数据
	MetaEnable bool
	// TempIndex 临时变量索引
	TempIndex int
	// IsTemp 是否为临时变量
	IsTemp bool
	// MetaStr 字符串类型的元数据
	MetaStr []string
}

// selectFromMeta 对整数切片执行常见的选择/丢弃操作
// 支持的模式：
//   - "kh": 保留最高的n个值
//   - "kl": 保留最低的n个值
//   - "dh": 丢弃最高的n个值并返回其余值
//   - "dl": 丢弃最低的n个值并返回其余值
//
// 返回选择的切片及其总和
func selectFromMeta(src []int, n int, mode string) ([]int, int) {
	if src == nil || len(src) == 0 {
		return []int{}, 0
	}

	arr := append([]int(nil), src...)
	switch mode {
	case "kh":
		sort.Slice(arr, func(i, j int) bool { return arr[i] > arr[j] })
		if n > len(arr) {
			n = len(arr)
		}
		sel := arr[:n]
		s := 0
		for _, v := range sel {
			s += v
		}
		return sel, s
	case "kl":
		sort.Ints(arr)
		if n > len(arr) {
			n = len(arr)
		}
		sel := arr[:n]
		s := 0
		for _, v := range sel {
			s += v
		}
		return sel, s
	case "dh":
		sort.Slice(arr, func(i, j int) bool { return arr[i] > arr[j] })
		if n >= len(arr) {
			return []int{}, 0
		}
		sel := arr[n:]
		s := 0
		for _, v := range sel {
			s += v
		}
		return sel, s
	case "dl":
		sort.Ints(arr)
		if n >= len(arr) {
			return []int{}, 0
		}
		sel := arr[n:]
		s := 0
		for _, v := range sel {
			s += v
		}
		return sel, s
	default:
		return []int{}, 0
	}
}

// resolveMetaValues 将可能包含Meta或MetaStr的Value转换为整数切片
// 成功时返回解析的切片和true，失败时返回nil和false
func (r *RD) resolveMetaValues(v Value) ([]int, bool) {
	if !v.MetaEnable {
		return []int{v.V}, true
	}

	if v.Meta != nil {
		return append([]int(nil), v.Meta...), true
	}

	if v.MetaStr != nil && len(v.MetaStr) > 0 {
		data := make([]interface{}, len(v.MetaStr))
		for i, s := range v.MetaStr {
			data[i] = s
		}
		res := r.getFromMetaTuple(data, false, true)
		if len(res) == len(data) {
			return res, true
		}
	}

	return nil, false
}

// tokenize 将表达式分割为标记：数字、运算符、括号等
func tokenize(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	var toks []string
	i := 0

	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' {
			i++
			continue
		}

		// 支持双引号字符串字面量
		if c == '"' {
			j := i + 1
			var sb strings.Builder
			for j < len(s) {
				if s[j] == '\\' && j+1 < len(s) {
					sb.WriteByte(s[j+1])
					j += 2
					continue
				}
				if s[j] == '"' {
					break
				}
				sb.WriteByte(s[j])
				j++
			}
			if j >= len(s) || s[j] != '"' {
				return nil, fmt.Errorf("unterminated string literal")
			}
			toks = append(toks, "\""+sb.String()+"\"")
			i = j + 1
			continue
		}

		if isDigit(c) {
			j := i + 1
			for j < len(s) && isDigit(s[j]) {
				j++
			}
			toks = append(toks, s[i:j])
			i = j
			continue
		}

		// 括号元组：捕获整个 [...] 作为一个标记（支持嵌套）
		if c == '[' {
			depth := 0
			j := i
			for j < len(s) {
				if s[j] == '[' {
					depth++
				} else if s[j] == ']' {
					depth--
					if depth == 0 {
						break
					}
				} else if s[j] == '"' {
					k := j + 1
					for k < len(s) {
						if s[k] == '\\' && k+1 < len(s) {
							k += 2
							continue
						}
						if s[k] == '"' {
							break
						}
						k++
					}
					j = k
				}
				j++
			}
			if j >= len(s) || s[j] != ']' {
				return nil, fmt.Errorf("unterminated bracketed tuple")
			}
			toks = append(toks, s[i:j+1])
			i = j + 1
			continue
		}

		// 单字符运算符和标点符号
		if c == '+' || c == '-' || c == '*' || c == '/' || c == '^' || c == '(' || c == ')' || c == ',' || c == '?' || c == ':' || c == '=' || c == '<' || c == '>' || c == '&' || c == '|' || c == '%' {
			toks = append(toks, string(c))
			i++
			continue
		}

		// 支持字母运算符如 'd'
		if c == '$' {
			j := i + 1
			for j < len(s) && ((s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= '0' && s[j] <= '9')) {
				j++
			}
			toks = append(toks, s[i:j])
			i = j
			continue
		}

		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			j := i + 1
			for j < len(s) && ((s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z')) {
				j++
			}
			toks = append(toks, s[i:j])
			i = j
			continue
		}

		return nil, fmt.Errorf("unexpected char '%c'", c)
	}

	return toks, nil
}

// 运算符优先级映射
var prec = map[string]int{
	"|":   2,
	"&":   2,
	"<":   1,
	">":   1,
	"+":   3,
	"-":   3,
	"*":   4,
	"/":   4,
	"^":   5,
	"d":   7,
	"df":  7,
	"k":   6,
	"q":   6,
	"a":   7,
	"c":   7,
	"a_m": 7,
	"c_m": 7,
	"b":   7,
	"p":   7,
	"f":   7,
	"kh":  6,
	"kl":  6,
	"dh":  6,
	"dl":  6,
	"min": 6,
	"max": 6,
	"sp":  6,
	"tp":  6,
	"lp":  6,
	"?":   8,
	"=":   9,
}

// isLeftAssoc 判断运算符是否为左结合
func isLeftAssoc(op string) bool {
	if op == "^" || op == "=" {
		return false
	}
	return true
}

// isOperator 判断标记是否为运算符
func isOperator(tok string) bool {
	if _, ok := prec[tok]; ok {
		return true
	}
	return false
}

// preProcessTokens 处理特殊模式如：<left> a <threshold> m <faces>
// 并将它们重写为：<left> <threshold> <faces> a_m
// 以便RPN转换和评估器可以将`a_m`/`c_m`视为三元运算符
func preProcessTokens(toks []string, defaultD int) []string {
	// 两阶段规范化：
	// 1) 为某些运算符的缺失左右操作数插入合理的默认值
	// 2) 重写模式如：<left> a <threshold> m <faces> -> <left> <threshold> <faces> a_m

	// 阶段1
	out := make([]string, 0, len(toks)+4)
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		low := strings.ToLower(tok)

		switch low {
		case "d", "b", "p", "f", "df", "a", "c":
			needLeft := false
			if len(out) == 0 {
				needLeft = true
			} else {
				last := out[len(out)-1]
				if isOperator(strings.ToLower(last)) || last == "(" || last == "?" || last == ":" {
					needLeft = true
				}
			}

			if needLeft {
				switch low {
				case "d":
					out = append(out, "1")
				case "b", "p", "a", "c":
					out = append(out, "1")
				case "f", "df":
					out = append(out, "4")
				}
			}

			out = append(out, tok)

			needRight := false
			if i+1 >= len(toks) {
				needRight = true
			} else {
				next := toks[i+1]
				if isOperator(strings.ToLower(next)) || next == ")" || next == ":" {
					needRight = true
				}
			}

			if needRight {
				switch low {
				case "b", "p":
					out = append(out, "1")
				case "f", "df":
					out = append(out, "3")
				}
			}
		default:
			out = append(out, tok)
		}
	}

	// 阶段2：重写带有m的a/c为a_m/c_m
	res := make([]string, 0, len(out))
	i := 0
	for i < len(out) {
		if i+4 < len(out) {
			op := strings.ToLower(out[i+1])
			mid := strings.ToLower(out[i+3])
			if (op == "a" || op == "c") && mid == "m" {
				if _, err1 := strconv.Atoi(out[i+2]); err1 == nil {
					if _, err2 := strconv.Atoi(out[i+4]); err2 == nil {
						res = append(res, out[i])   // left
						res = append(res, out[i+2]) // threshold
						res = append(res, out[i+4]) // faces
						res = append(res, op+"_m")
						i += 5
						continue
					}
				}
			}
		}
		res = append(res, out[i])
		i++
	}

	// 额外处理：处理d%和d的默认右侧操作数
	final := make([]string, 0, len(res))
	j := 0
	for j < len(res) {
		if strings.ToLower(res[j]) == "d" {
			// 如果下一个标记是'%'，则视为100
			if j+1 < len(res) && res[j+1] == "%" {
				if j == 0 {
					final = append(final, "1")
				}
				final = append(final, "d")
				final = append(final, strconv.Itoa(100))
				j += 2
				continue
			}

			// 如果下一个标记是运算符或缺失，插入defaultD
			if j+1 >= len(res) || isOperator(strings.ToLower(res[j+1])) || res[j+1] == ")" || res[j+1] == ":" {
				if j == 0 {
					final = append(final, "1")
				}
				final = append(final, "d")
				final = append(final, strconv.Itoa(defaultD))
				j++
				continue
			}
		}
		final = append(final, res[j])
		j++
	}

	return final
}

// toRPN 使用调度场算法将标记转换为逆波兰表示法
func toRPN(tokens []string) ([]string, error) {
	var out []string
	var stack []string

	for _, tok := range tokens {
		if _, err := strconv.Atoi(tok); err == nil {
			out = append(out, tok)
			continue
		}

		// 允许临时变量标记如$t1作为操作数
		if strings.HasPrefix(tok, "$") {
			out = append(out, tok)
			continue
		}

		// 允许括号元组标记作为操作数
		if len(tok) > 0 && tok[0] == '[' {
			out = append(out, tok)
			continue
		}

		// 允许非注册运算符的裸标识符作为操作数
		if !isOperator(strings.ToLower(tok)) && len(tok) > 0 && ((tok[0] >= 'a' && tok[0] <= 'z') || (tok[0] >= 'A' && tok[0] <= 'Z')) {
			out = append(out, tok)
			continue
		}

		// 允许双引号字符串字面量作为操作数
		if len(tok) >= 2 && tok[0] == '"' && tok[len(tok)-1] == '"' {
			out = append(out, tok)
			continue
		}

		if isOperator(strings.ToLower(tok)) {
			op := strings.ToLower(tok)
			if op == "df" {
				op = "f"
			}

			for len(stack) > 0 {
				top := stack[len(stack)-1]
				if isOperator(top) && ((isLeftAssoc(op) && prec[op] <= prec[top]) || (!isLeftAssoc(op) && prec[op] < prec[top])) {
					out = append(out, top)
					stack = stack[:len(stack)-1]
				} else {
					break
				}
			}
			stack = append(stack, op)
			continue
		}

		// 三元运算符'?'和':'的特殊处理
		if tok == "?" {
			stack = append(stack, "?")
			continue
		}

		if tok == ":" {
			found := false
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				if top == "?" {
					stack = stack[:len(stack)-1]
					stack = append(stack, ":")
					found = true
					break
				}
				out = append(out, top)
				stack = stack[:len(stack)-1]
			}
			if !found {
				return nil, fmt.Errorf("mismatched ternary ':'")
			}
			continue
		}

		if tok == "(" {
			stack = append(stack, tok)
			continue
		}

		if tok == ")" {
			found := false
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if top == "(" {
					found = true
					break
				}
				out = append(out, top)
			}
			if !found {
				return nil, fmt.Errorf("mismatched parentheses")
			}
			continue
		}

		// 未知标记
		return nil, fmt.Errorf("unknown token %s", tok)
	}

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if top == "(" || top == ")" {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		out = append(out, top)
	}

	return out, nil
}

// evalRPN 评估RPN标记；支持基本运算和使用RNG的'd'运算符
func (r *RD) evalRPN(rpn []string) (Value, ErrorType) {
	var st []Value
	push := func(v Value) { st = append(st, v) }
	pop := func() (Value, bool) {
		if len(st) == 0 {
			return Value{}, false
		}
		v := st[len(st)-1]
		st = st[:len(st)-1]
		return v, true
	}

	for _, tok := range rpn {
		if v, err := strconv.Atoi(tok); err == nil {
			push(Value{V: v, Meta: nil, MetaEnable: false})
			continue
		}

		// 括号元组字面量标记如[a,b,c]
		if len(tok) >= 2 && tok[0] == '[' && tok[len(tok)-1] == ']' {
			inner := tok[1 : len(tok)-1]
			elems := make([]string, 0)
			sb := strings.Builder{}
			depth := 0
			inStr := false

			for i := 0; i < len(inner); i++ {
				ch := inner[i]
				if ch == '"' {
					inStr = !inStr
					sb.WriteByte(ch)
					continue
				}
				if inStr {
					sb.WriteByte(ch)
					continue
				}
				if ch == '(' || ch == '[' {
					depth++
				} else if ch == ')' || ch == ']' {
					depth--
				}
				if ch == ',' && depth == 0 {
					elems = append(elems, strings.TrimSpace(sb.String()))
					sb.Reset()
					continue
				}
				sb.WriteByte(ch)
			}
			if sb.Len() > 0 {
				elems = append(elems, strings.TrimSpace(sb.String()))
			}

			metaInts := make([]int, 0, len(elems))
			metaStrs := make([]string, 0, len(elems))
			for _, el := range elems {
				if el == "" {
					continue
				}
				if vi, err := strconv.Atoi(el); err == nil {
					metaInts = append(metaInts, vi)
				} else {
					metaStrs = append(metaStrs, el)
				}
			}

			if len(metaStrs) > 0 && len(metaInts) > 0 {
				all := make([]string, 0, len(elems))
				for _, el := range elems {
					all = append(all, el)
				}
				push(Value{V: 0, Meta: nil, MetaEnable: true, MetaStr: all})
				continue
			}

			if len(metaStrs) > 0 {
				push(Value{V: 0, Meta: nil, MetaEnable: true, MetaStr: metaStrs})
				continue
			}

			// 全部是整数
			push(Value{V: 0, Meta: metaInts, MetaEnable: true})
			continue
		}

		// 字符串字面量
		if len(tok) >= 2 && tok[0] == '"' && tok[len(tok)-1] == '"' {
			content := tok[1 : len(tok)-1]
			push(Value{V: 0, Meta: nil, MetaEnable: true, MetaStr: []string{content}})
			continue
		}

		// 临时变量检索标记如$t或$t2
		if strings.HasPrefix(tok, "$") {
			idx := 1
			if len(tok) > 2 {
				if n, err := strconv.Atoi(tok[2:]); err == nil {
					idx = n
				}
			}

			val := 0
			found := false
			if r.temp != nil {
				if vv, ok := r.temp[idx]; ok {
					val = vv
					found = true
				}
			}
			if !found && r.ValueTable != nil {
				key := strings.ToUpper(fmt.Sprintf("t%d", idx))
				if vv, ok := r.ValueTable[key]; ok {
					val = vv
					found = true
				}
				if !found {
					key2 := fmt.Sprintf("t%d", idx)
					if vv, ok := r.ValueTable[key2]; ok {
						val = vv
						found = true
					}
				}
			}

			push(Value{V: val, TempIndex: idx, IsTemp: true})
			continue
		}

		switch tok {
		case ":":
			// 三元运算符在RPN中：弹出false, 弹出true, 弹出条件
			c, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			b, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			a, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}

			if a.V != 0 {
				push(b)
			} else {
				push(c)
			}
			continue
		case "+":
			b, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			a, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			push(Value{V: a.V + b.V})
		case "-":
			b, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			a, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			push(Value{V: a.V - b.V})
		case "*":
			b, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			a, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			push(Value{V: a.V * b.V})
		case "/":
			b, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			a, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if b.V == 0 {
				return Value{}, ErrNodeRightValInvalid
			}
			push(Value{V: a.V / b.V})
		case ">": // 大于比较
			bgt, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			agt, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if agt.V > bgt.V {
				push(Value{V: 1})
			} else {
				push(Value{V: 0})
			}
		case "<": // 小于比较
			bgt, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			agt, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if agt.V < bgt.V {
				push(Value{V: 1})
			} else {
				push(Value{V: 0})
			}
		case "&": // 按位与
			bbit, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			abit, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			push(Value{V: abit.V & bbit.V})
		case "|": // 按位或
			bbit2, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			abit2, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			push(Value{V: abit2.V | bbit2.V})
		case "^":
			b, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			a, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if a.V == 0 && b.V == 0 {
				return Value{}, ErrNodeLeftValInvalid
			}
			if b.V < 0 {
				return Value{}, ErrNodeRightValInvalid
			}
			res := 1
			for i := 0; i < b.V; i++ {
				res *= a.V
			}
			push(Value{V: res})
		case "d": // 掷骰运算符
			sidesV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			timesV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}

			var sides int
			if sidesV.MetaEnable && len(sidesV.Meta) > 0 {
				sides = sidesV.Meta[len(sidesV.Meta)-1]
			} else {
				sides = sidesV.V
			}

			var times int
			if timesV.MetaEnable && len(timesV.Meta) > 0 {
				times = timesV.Meta[len(timesV.Meta)-1]
			} else {
				times = timesV.V
			}

			if times <= 0 || times > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}
			if sides <= 0 || sides > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}

			rolls := make([]int, 0, times)
			sum := 0
			for i := 0; i < times; i++ {
				rnum := r.rng.Intn(sides) + 1
				rolls = append(rolls, rnum)
				sum += rnum
			}

			push(Value{V: sum, Meta: rolls, MetaEnable: true})
		case "k": // 保留最高k个
			param, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			left, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			k := param.V
			if k <= 0 {
				return Value{}, ErrNodeRightValInvalid
			}

			rolls, ok := r.resolveMetaValues(left)
			if !ok {
				return Value{}, ErrNodeLeftValInvalid
			}
			sel, s := selectFromMeta(rolls, k, "kh")
			push(Value{V: s, Meta: sel, MetaEnable: len(sel) > 0})
		case "a": // 附加链：掷times组m面骰子；任何大于等于threshold的结果都会添加到下一轮
			rightV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			times := leftV.V
			threshold := rightV.V
			if times < 0 || times > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}
			if threshold <= 0 || threshold > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}

			m := 10 // 'a'的默认面数
			total := 0
			meta := []int{}
			nextCount := times

			for nextCount > 0 {
				cur := nextCount
				nextCount = 0
				for i := 0; i < cur; i++ {
					rnum := r.rng.Intn(m) + 1
					meta = append(meta, rnum)
					if rnum >= threshold {
						nextCount++
					}
					if rnum >= threshold {
						total++
					}
				}
				if len(meta) > 10000 {
					break
				}
			}

			push(Value{V: total, Meta: meta, MetaEnable: len(meta) > 0})
		case "a_m": // 三元运算符：左侧times，threshold，faces
			facesV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			rightV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			times := leftV.V
			threshold := rightV.V
			m := facesV.V
			if times < 0 || times > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}
			if threshold <= 0 || threshold > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}
			if m <= 0 || m > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}

			total := 0
			meta := []int{}
			nextCount := times

			for nextCount > 0 {
				cur := nextCount
				nextCount = 0
				for i := 0; i < cur; i++ {
					rnum := r.rng.Intn(m) + 1
					meta = append(meta, rnum)
					if rnum >= threshold {
						nextCount++
					}
					if rnum >= threshold {
						total++
					}
				}
				if len(meta) > 10000 {
					break
				}
			}

			push(Value{V: total, Meta: meta, MetaEnable: len(meta) > 0})
		case "c": // 压缩链：掷组并求和每轮的最大值；只要有任何掷骰结果>=threshold就继续
			rightC, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftC, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			timesC := leftC.V
			thresholdC := rightC.V
			if timesC < 0 || timesC > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}
			if thresholdC <= 0 || thresholdC > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}

			mC := 10
			totalC := 0
			metaC := []int{}
			nextC := timesC

			for nextC > 0 {
				cur := nextC
				nextC = 0
				maxv := 0
				for i := 0; i < cur; i++ {
					rnum := r.rng.Intn(mC) + 1
					metaC = append(metaC, rnum)
					if rnum > maxv {
						maxv = rnum
					}
					if rnum >= thresholdC {
						nextC++
					}
				}
				totalC += maxv
				if len(metaC) > 10000 {
					break
				}
			}

			push(Value{V: totalC, Meta: metaC, MetaEnable: len(metaC) > 0})
		case "c_m": // 三元运算符：左侧times，threshold，faces
			facesV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			rightV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftV, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			timesC := leftV.V
			thresholdC := rightV.V
			mC := facesV.V
			if timesC < 0 || timesC > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}
			if thresholdC <= 0 || thresholdC > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}
			if mC <= 0 || mC > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}

			totalC := 0
			metaC := []int{}
			nextC := timesC

			for nextC > 0 {
				cur := nextC
				nextC = 0
				maxv := 0
				for i := 0; i < cur; i++ {
					rnum := r.rng.Intn(mC) + 1
					metaC = append(metaC, rnum)
					if rnum > maxv {
						maxv = rnum
					}
					if rnum >= thresholdC {
						nextC++
					}
				}
				totalC += maxv
				if len(metaC) > 10000 {
					break
				}
			}

			push(Value{V: totalC, Meta: metaC, MetaEnable: len(metaC) > 0})
		case "b": // 奖励机制(COC)：将d100作为两个d10（十位和个位，0..9）投掷
			// 然后投掷paramB个额外的d10（0..9）并将十位数字替换为额外骰子中的最小值
			// 如果十位和个位都是0，结果是100
			paramB, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftB, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if paramB.V < 0 {
				return Value{}, ErrNodeRightValInvalid
			}
			if paramB.V > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}
			if leftB.V > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}

			tens := r.rng.Intn(10)
			units := r.rng.Intn(10)
			rolls := make([]int, 0, paramB.V)
			for i := 0; i < paramB.V; i++ {
				rr := r.rng.Intn(10)
				rolls = append(rolls, rr)
			}

			var out int
			if tens == 0 && units == 0 {
				out = 100
			} else {
				if len(rolls) > 0 {
					mn := rolls[0]
					for _, v := range rolls[1:] {
						if v < mn {
							mn = v
						}
					}
					tens = mn
				}
				out = tens*10 + units
			}

			meta := make([]int, 0, 2+len(rolls))
			meta = append(meta, tens, units)
			meta = append(meta, rolls...)
			push(Value{V: out, Meta: meta, MetaEnable: len(meta) > 0})
		case "p": // 惩罚机制(COC)：与奖励相同，但用额外骰子的最大值替换十位
			paramP, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftP, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if paramP.V < 0 {
				return Value{}, ErrNodeRightValInvalid
			}
			if paramP.V > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}
			if leftP.V > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}

			tens := r.rng.Intn(10)
			units := r.rng.Intn(10)
			rollsP := make([]int, 0, paramP.V)
			for i := 0; i < paramP.V; i++ {
				rr := r.rng.Intn(10)
				rollsP = append(rollsP, rr)
			}

			var outP int
			if tens == 0 && units == 0 {
				outP = 100
			} else {
				if len(rollsP) > 0 {
					mx := rollsP[0]
					for _, v := range rollsP[1:] {
						if v > mx {
							mx = v
						}
					}
					tens = mx
				}
				outP = tens*10 + units
			}

			metaP := make([]int, 0, 2+len(rollsP))
			metaP = append(metaP, tens, units)
			metaP = append(metaP, rollsP...)
			push(Value{V: outP, Meta: metaP, MetaEnable: len(metaP) > 0})
		case "=": // 赋值：弹出右侧值然后左侧占位符
			rightA, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftA, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if !leftA.IsTemp {
				return Value{}, ErrNodeLeftValInvalid
			}

			if r.temp == nil {
				r.temp = map[int]int{}
			}
			r.temp[leftA.TempIndex] = rightA.V

			if r.ValueTable == nil {
				r.ValueTable = map[string]int{}
			}
			tkey := strings.ToUpper(fmt.Sprintf("t%d", leftA.TempIndex))
			r.ValueTable[tkey] = rightA.V

			push(Value{V: rightA.V})
		case "lp": // 重复/循环运算符：左侧元数据列表重复右侧次数
			paramLp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftLp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			timesLp := paramLp.V
			if timesLp <= 0 {
				return Value{}, ErrNodeRightValInvalid
			}

			if leftLp.MetaStr != nil && len(leftLp.MetaStr) > 0 {
				templates := leftLp.MetaStr
				outList := make([]string, 0, len(templates)*timesLp)
				idx := 1
				for t := 0; t < timesLp; t++ {
					for _, tmpl := range templates {
						s := strings.ReplaceAll(tmpl, "{i}", strconv.Itoa(idx))
						outList = append(outList, s)
						idx++
					}
				}
				push(Value{V: 0, Meta: nil, MetaEnable: true, MetaStr: outList})
				continue
			}

			rollsLp, ok := r.resolveMetaValues(leftLp)
			if !ok {
				return Value{}, ErrNodeLeftValInvalid
			}
			newList := make([]int, 0, len(rollsLp)*timesLp)
			for i := 0; i < timesLp; i++ {
				newList = append(newList, rollsLp...)
			}
			sumLp := 0
			for _, vv := range newList {
				sumLp += vv
			}
			push(Value{V: sumLp, Meta: newList, MetaEnable: len(newList) > 0})
		case "q": // 保留最低q个
			param, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			left, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			q := param.V
			if q <= 0 {
				return Value{}, ErrNodeRightValInvalid
			}

			rolls, ok := r.resolveMetaValues(left)
			if !ok {
				return Value{}, ErrNodeLeftValInvalid
			}
			sel, s := selectFromMeta(rolls, q, "kl")
			push(Value{V: s, Meta: sel, MetaEnable: len(sel) > 0})
		case "kh", "kl", "dh", "dl":
			// 弹出参数然后左侧
			paramOp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftOp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			n := paramOp.V
			if n <= 0 {
				return Value{}, ErrNodeRightValInvalid
			}

			rollsRaw, ok := r.resolveMetaValues(leftOp)
			if !ok {
				return Value{}, ErrNodeLeftValInvalid
			}
			if len(rollsRaw) == 0 {
				return Value{}, ErrNodeLeftValInvalid
			}

			sel, sum := selectFromMeta(rollsRaw, n, tok)
			push(Value{V: sum, Meta: sel, MetaEnable: len(sel) > 0})
		case "min", "max":
			paramOp2, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftOp2, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			n2 := paramOp2.V
			if n2 <= 0 {
				return Value{}, ErrNodeRightValInvalid
			}

			rollsRaw2 := leftOp2.Meta
			if !leftOp2.MetaEnable {
				rollsRaw2 = []int{leftOp2.V}
			}

			resList := make([]int, len(rollsRaw2))
			sum2 := 0
			for i, rv := range rollsRaw2 {
				if tok == "max" {
					if rv > n2 {
						rv = n2
					}
				} else {
					if rv < n2 {
						rv = n2
					}
				}
				resList[i] = rv
				sum2 += rv
			}

			push(Value{V: sum2, Meta: resList, MetaEnable: true})
		case "f": // fudge/fate骰子：左侧次数掷出[-1,1]，求和
			rightF, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftF, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			if rightF.V <= 1 || rightF.V > 10000 {
				return Value{}, ErrNodeRightValInvalid
			}
			if leftF.V <= 0 || leftF.V > 10000 {
				return Value{}, ErrNodeLeftValInvalid
			}

			rollsF := make([]int, 0, leftF.V)
			sumF := 0
			for i := 0; i < leftF.V; i++ {
				rnum := r.rng.Intn(3) - 1
				rollsF = append(rollsF, rnum)
				sumF += rnum
			}

			push(Value{V: sumF, Meta: rollsF, MetaEnable: true})
		case "sp": // 选择位置：弹出参数然后左侧；返回指定位置的单个元素
			paramSp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftSp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			idx := paramSp.V

			rollsSp := leftSp.Meta
			if !leftSp.MetaEnable {
				if idx == 1 || idx == -1 {
					val := leftSp.V
					push(Value{V: val, Meta: []int{val}, MetaEnable: true})
					continue
				}
				return Value{}, ErrNodeLeftValInvalid
			}

			if idx == 0 {
				return Value{}, ErrNodeRightValInvalid
			}
			var pos int
			if idx > 0 {
				pos = idx - 1
			} else {
				pos = len(rollsSp) + idx
			}
			if pos < 0 || pos >= len(rollsSp) {
				return Value{}, ErrNodeRightValInvalid
			}

			v := rollsSp[pos]
			push(Value{V: v, Meta: []int{v}, MetaEnable: true})
		case "tp": // 取得位置：弹出参数然后左侧；移除指定位置的元素并返回剩余元素的总和
			paramTp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			leftTp, ok := pop()
			if !ok {
				return Value{}, ErrNodeStackEmpty
			}
			idx2 := paramTp.V

			rollsTp := leftTp.Meta
			if !leftTp.MetaEnable {
				if idx2 == 1 || idx2 == -1 {
					push(Value{V: 0, Meta: []int{}, MetaEnable: false})
					continue
				}
				return Value{}, ErrNodeLeftValInvalid
			}

			if idx2 == 0 {
				return Value{}, ErrNodeRightValInvalid
			}
			var pos2 int
			if idx2 > 0 {
				pos2 = idx2 - 1
			} else {
				pos2 = len(rollsTp) + idx2
			}
			if pos2 < 0 || pos2 >= len(rollsTp) {
				return Value{}, ErrNodeRightValInvalid
			}

			newList := append([]int{}, rollsTp[:pos2]...)
			if pos2+1 < len(rollsTp) {
				newList = append(newList, rollsTp[pos2+1:]...)
			}
			sumTp := 0
			for _, vv := range newList {
				sumTp += vv
			}
			push(Value{V: sumTp, Meta: newList, MetaEnable: len(newList) > 0})
		default:
			return Value{}, ErrUnknownGenerate
		}
	}

	if len(st) != 1 {
		return Value{}, ErrUnknownGenerate
	}

	return st[0], ""
}

// evalTokens 评估标记切片并支持短路三元运算符?:
// 通过定位顶级'?'并匹配':'来实现短路；非三元切片通过转换为RPN并使用evalRPN进行评估
func (r *RD) evalTokens(tokens []string) (Value, ErrorType) {
	// First, evaluate innermost parenthesized subexpressions so ternaries inside
	// parentheses can be handled with short-circuiting. We replace each '( ... )'
	// with its evaluated integer value token and keep side-effects (temp writes).
	for {
		// find a closing paren whose matching open is NOT immediately after a '?' or ':'
		closeIdx := -1
		var openIdx int
		found := false
		for i := 0; i < len(tokens); i++ {
			if tokens[i] != ")" {
				continue
			}
			// find matching open for this close
			d := 0
			oi := -1
			for k := i; k >= 0; k-- {
				if tokens[k] == ")" {
					d++
				}
				if tokens[k] == "(" {
					d--
				}
				if d == 0 {
					oi = k
					break
				}
			}
			if oi == -1 {
				return Value{}, ErrUnknownGenerate
			}
			// if '(' is immediately preceded by '?' or ':' then this paren likely
			// is a branch of a ternary; skip it to avoid evaluating both branches
			if oi > 0 {
				if tokens[oi-1] == "?" || tokens[oi-1] == ":" {
					continue
				}
			}
			closeIdx = i
			openIdx = oi
			found = true
			break
		}
		if !found {
			break
		}
		// evaluate inside
		inner := append([]string(nil), tokens[openIdx+1:closeIdx]...)
		v, derr := r.evalTokens(inner)
		if derr != "" {
			return Value{}, derr
		}
		// replace tokens[openIdx:closeIdx+1] with the integer token of v.V
		sval := strconv.Itoa(v.V)
		newTok := make([]string, 0, len(tokens)-(closeIdx-openIdx))
		newTok = append(newTok, tokens[:openIdx]...)
		newTok = append(newTok, sval)
		if closeIdx+1 < len(tokens) {
			newTok = append(newTok, tokens[closeIdx+1:]...)
		}
		tokens = newTok
	}

	// Trim outer parentheses that enclose the entire token list
	for {
		if len(tokens) >= 2 && tokens[0] == "(" {
			// find matching ) for tokens[0]
			d := 0
			match := -1
			for i := 0; i < len(tokens); i++ {
				if tokens[i] == "(" {
					d++
				}
				if tokens[i] == ")" {
					d--
				}
				if d == 0 {
					match = i
					break
				}
			}
			if match == len(tokens)-1 {
				// outer parentheses wrap entire expression: strip and continue
				tokens = tokens[1 : len(tokens)-1]
				continue
			}
		}
		break
	}

	// find top-level '?'
	depth := 0
	for i, tok := range tokens {
		if tok == "(" {
			depth++
			continue
		}
		if tok == ")" {
			depth--
			continue
		}
		if tok == "?" && depth == 0 {
			// found ternary operator at i; find matching ':'
			qcount := 1
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j] == "(" {
					depth++
				} else if tokens[j] == ")" {
					depth--
				}
				if tokens[j] == "?" && depth == 0 {
					qcount++
				} else if tokens[j] == ":" && depth == 0 {
					qcount--
					if qcount == 0 {
						// split into cond ? true : false
						condToks := tokens[:i]
						trueToks := tokens[i+1 : j]
						falseToks := tokens[j+1:]
						// evaluate condition
						condVal, derr := r.evalTokens(condToks)
						if derr != "" {
							return Value{}, derr
						}
						if condVal.V != 0 {
							return r.evalTokens(trueToks)
						}
						return r.evalTokens(falseToks)
					}
				}
			}
			return Value{}, ErrUnknownGenerate
		}
	}
	// no top-level ternary: fallback to RPN evaluation
	// preprocess tokens with RD defaults (e.g., default d faces) before RPN
	tokens = preProcessTokens(tokens, r.DefaultFaces)
	rpn, err := toRPN(tokens)
	if err != nil {
		return Value{}, ErrUnknownGenerate
	}
	return r.evalRPN(rpn)
}
