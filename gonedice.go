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

// ErrorType mirrors some of the Python error categories (simplified)
type ErrorType string

const (
    ErrUnknownGenerate     ErrorType = "UNKNOWN_GENERATE_FATAL"
    ErrInputRawInvalid     ErrorType = "INPUT_RAW_INVALID"
    ErrNodeStackEmpty      ErrorType = "NODE_STACK_EMPTY"
    ErrNodeLeftValInvalid  ErrorType = "NODE_LEFT_VAL_INVALID"
    ErrNodeRightValInvalid ErrorType = "NODE_RIGHT_VAL_INVALID"
)

// Result holds a single roll result
type Result struct {
    Value       int
    Min         int
    Max         int
    Detail      string
    MetaTuple   []interface{}
    Error       ErrorType
}

// RD is the runner similar to Python RD
type RD struct {
    Expr       string
    origin     string
    ValueTable map[string]int
    rng        *rand.Rand
    res        Result
    temp       map[int]int
}

// New creates a new RD
func New(expr string, valueTable map[string]int) *RD {
    src := expr
    if valueTable != nil {
        // do case-insensitive replacement later
    }
    return &RD{
        Expr:       expr,
        origin:     strings.ToLower(src),
        ValueTable: valueTable,
        rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
        temp:       map[int]int{},
    }
}

// Roll evaluates the expression and populates Result. This is a simplified, incremental
// port: supports numbers, + - * / ^, parentheses, variable replacement {VAR} and basic d (NdM) dice.
func (r *RD) Roll() {
    // first replace variables like {KEY}
    expr, err := r.replaceVars(r.origin)
    if err != nil {
        r.res.Error = ErrInputRawInvalid
        return
    }
    // tokenize
    tokens, terr := tokenize(expr)
    if terr != nil {
        r.res.Error = ErrInputRawInvalid
        return
    }
    // evaluate tokens with ternary-shortcircuit aware evaluator
    val, derr := r.evalTokens(tokens)
    if derr != "" {
        r.res.Error = derr
        return
    }
    r.res.Value = val.V
    r.res.Min = val.V
    r.res.Max = val.V
    // build a more detailed human-readable detail string
    r.res.Detail = r.buildDetail(val)
    // convert meta ints to interface{} slice for compatibility
    if val.MetaEnable {
        if val.MetaStr != nil && len(val.MetaStr) > 0 {
            meta := make([]interface{}, len(val.MetaStr))
            for i, vv := range val.MetaStr { meta[i] = vv }
            r.res.MetaTuple = meta
        } else {
            meta := make([]interface{}, len(val.Meta))
            for i, vv := range val.Meta { meta[i] = vv }
            // attempt to resolve any meta entries that are expressions (strings)
            r.res.MetaTuple = meta
            // pre-evaluate string entries to ints for convenience
            resolved := r.getFromMetaTuple(meta, false, true)
            if len(resolved) == len(meta) {
                // replace MetaTuple with resolved ints
                meta2 := make([]interface{}, len(resolved))
                for i, v := range resolved { meta2[i] = v }
                r.res.MetaTuple = meta2
            }
        }
    }
    r.res.Error = ""
}

// buildDetail constructs a readable representation including value, meta list (ints or strings),
// and optionally a snapshot of temp and ValueTable for debugging.
func (r *RD) buildDetail(val Value) string {
    parts := []string{}
    // value
    parts = append(parts, fmt.Sprintf("%d", val.V))
    // meta
    if val.MetaEnable {
        if val.MetaStr != nil && len(val.MetaStr) > 0 {
            // format as ["a","b"]
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
    // min/max
    if val.V != 0 {
        // include min/max if available and differ (currently they may be same)
        if r.res.Min != r.res.Max {
            parts = append(parts, fmt.Sprintf("min=%d", r.res.Min))
            parts = append(parts, fmt.Sprintf("max=%d", r.res.Max))
        }
    }
    // temp map snapshot
    if r.temp != nil && len(r.temp) > 0 {
        keys := make([]int, 0, len(r.temp))
        for k := range r.temp { keys = append(keys, k) }
        sort.Ints(keys)
        kvs := make([]string, 0, len(keys))
        for _, k := range keys { kvs = append(kvs, fmt.Sprintf("t%d=%d", k, r.temp[k])) }
        parts = append(parts, fmt.Sprintf("temp:{%s}", strings.Join(kvs, ",")))
    }
    // ValueTable snapshot
    if r.ValueTable != nil && len(r.ValueTable) > 0 {
        keys := make([]string, 0, len(r.ValueTable))
        for k := range r.ValueTable { keys = append(keys, k) }
        sort.Strings(keys)
        kvs := make([]string, 0, len(keys))
        for _, k := range keys { kvs = append(kvs, fmt.Sprintf("%s=%d", k, r.ValueTable[k])) }
        parts = append(parts, fmt.Sprintf("vt:{%s}", strings.Join(kvs, ",")))
    }
    return strings.Join(parts, " ")
}

// Result returns the computed result
func (r *RD) Result() Result {
    return r.res
}

// getFromMetaTuple evaluates a slice of meta elements that may be ints or string expressions.
// flagLast: if true, propagate last sub-result fields (res.Value, Detail etc.) into the caller's Result
// flagUpdate: if true, merge sub-RD ValueTable into caller's ValueTable when available
// Returns a slice of ints for successfully evaluated elements. On any failure returns empty slice.
func (r *RD) getFromMetaTuple(data []interface{}, flagLast bool, flagUpdate bool) []int {
    res := make([]int, 0, len(data))
    for _, el := range data {
        switch v := el.(type) {
        case int:
            res = append(res, v)
        case string:
            // decide what ValueTable to pass to sub RD:
            // - if flagUpdate is true, allow sub to see and modify parent's map
            // - if flagUpdate is false, give sub a shallow copy so its writes don't affect parent
            var subVT map[string]int
            if flagUpdate {
                subVT = r.ValueTable
            } else if r.ValueTable != nil {
                subVT = make(map[string]int, len(r.ValueTable))
                for kk, vv := range r.ValueTable { subVT[kk] = vv }
            } else {
                subVT = nil
            }
            sub := New(v, subVT)
            // share RNG for deterministic sequences
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
                    // propagate some fields
                    r.res.Value = sub.res.Value
                    r.res.Min = sub.res.Min
                    r.res.Max = sub.res.Max
                    r.res.Detail = sub.res.Detail
                    r.res.Error = sub.res.Error
                }
            } else {
                // on error, stop resolving
                return []int{}
            }
        default:
            // unsupported type: abort
            return []int{}
        }
    }
    return res
}

var varRe = regexp.MustCompile(`\{([^}]+)\}`)

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

// --- tokenization and parsing (shunting-yard) ---

func isDigit(ch byte) bool {
    return ch >= '0' && ch <= '9'
}

// Value holds a runtime value with optional meta (individual dice)
type Value struct {
    V int
    Meta []int
    MetaEnable bool
    TempIndex int
    IsTemp bool
    MetaStr []string
}

// tokenize splits expression into tokens: numbers, operators, parentheses
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
        // support double-quoted string literals
        if c == '"' {
            j := i + 1
            var sb strings.Builder
            for j < len(s) {
                if s[j] == '\\' && j+1 < len(s) {
                    // escape sequence
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
            // store token with quotes included so parser can recognize it
            toks = append(toks, "\""+sb.String()+"\"")
            i = j+1
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
        // multi-char operators: we support '^' '+' '-' '*' '/' and 'd'
        if c == '+' || c == '-' || c == '*' || c == '/' || c == '^' || c == '(' || c == ')' || c == '[' || c == ']' || c == ',' || c == '?' || c == ':' || c == '=' {
            toks = append(toks, string(c))
            i++
            continue
        }
        // support letter operators like 'd'
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

var prec = map[string]int{
    "|": 2,
    "&": 2,
    "+": 3,
    "-": 3,
    "*": 4,
    "/": 4,
    "^": 5,
    "d": 7,
    "df": 7,
    "k": 6,
    "q": 6,
    "a": 7,
    "c": 7,
    "b": 7,
    "p": 7,
    "f": 7,
    "kh": 6,
    "kl": 6,
    "dh": 6,
    "dl": 6,
    "min": 6,
    "max": 6,
    "sp": 6,
    "tp": 6,
    "lp": 6,
    "?": 8,
    "=": 9,
}

func isLeftAssoc(op string) bool {
    if op == "^" || op == "=" {
        return false
    }
    return true
}

func isOperator(tok string) bool {
    if _, ok := prec[tok]; ok {
        return true
    }
    return false
}

// toRPN converts tokens to Reverse Polish Notation using shunting-yard
func toRPN(tokens []string) ([]string, error) {
    var out []string
    var stack []string
    for _, tok := range tokens {
        if _, err := strconv.Atoi(tok); err == nil {
            out = append(out, tok)
            continue
        }
        // allow temporary var tokens like $t1 as operands
        if strings.HasPrefix(tok, "$") {
            out = append(out, tok)
            continue
        }
        // allow bare identifiers as operands if not registered operators (e.g., future tuple strings)
        if !isOperator(strings.ToLower(tok)) && len(tok) > 0 && ((tok[0] >= 'a' && tok[0] <= 'z') || (tok[0] >= 'A' && tok[0] <= 'Z')) {
            out = append(out, tok)
            continue
        }
            // allow double-quoted string literals as operands
            if len(tok) >= 2 && tok[0] == '"' && tok[len(tok)-1] == '"' {
                out = append(out, tok)
                continue
            }
        if isOperator(strings.ToLower(tok)) {
            op := strings.ToLower(tok)
            // map df to f (alias)
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
        // special handling for ternary ':' and '?'
        if tok == "?" {
            stack = append(stack, "?")
            continue
        }
        if tok == ":" {
            // pop until matching '?'
            found := false
            for len(stack) > 0 {
                top := stack[len(stack)-1]
                if top == "?" {
                    // pop '?'
                    stack = stack[:len(stack)-1]
                    // push ':' as placeholder operator
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
        // unknown token
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

// evalRPN evaluates RPN tokens; supports basic ops and 'd' operator using RNG
func (r *RD) evalRPN(rpn []string) (Value, ErrorType) {
    var st []Value
    push := func(v Value) { st = append(st, v) }
    pop := func() (Value, bool) {
        if len(st) == 0 { return Value{}, false }
        v := st[len(st)-1]
        st = st[:len(st)-1]
        return v, true
    }

    for _, tok := range rpn {
        if v, err := strconv.Atoi(tok); err == nil {
            push(Value{V: v, Meta: nil, MetaEnable: false})
            continue
        }
        // string literal
        if len(tok) >= 2 && tok[0] == '"' && tok[len(tok)-1] == '"' {
            // unquote (we stored unescaped content inside)
            content := tok[1:len(tok)-1]
            push(Value{V: 0, Meta: nil, MetaEnable: true, MetaStr: []string{content}})
            continue
        }
        // temporary var retrieval token like $t or $t2
        if strings.HasPrefix(tok, "$") {
            // parse index number after $t, default 1
            idx := 1
            if len(tok) > 2 {
                if n, err := strconv.Atoi(tok[2:]); err == nil {
                    idx = n
                }
            }
            val := 0
            found := false
            if r.temp != nil {
                if vv, ok := r.temp[idx]; ok { val = vv; found = true }
            }
            if !found && r.ValueTable != nil {
                // check uppercase key like T1
                key := strings.ToUpper(fmt.Sprintf("t%d", idx))
                if vv, ok := r.ValueTable[key]; ok { val = vv; found = true }
                if !found {
                    // also check lowercase just in case
                    key2 := fmt.Sprintf("t%d", idx)
                    if vv, ok := r.ValueTable[key2]; ok { val = vv; found = true }
                }
            }
            push(Value{V: val, TempIndex: idx, IsTemp: true})
            continue
        }
        switch tok {
        case ":":
            // ternary operator in RPN: pop false, pop true, pop cond
            c, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            b, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            a, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if a.V != 0 {
                push(b)
            } else {
                push(c)
            }
            continue
        case "+":
            b, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            a, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            res := a.V + b.V
            // combine meta: if either has meta lists, keep simple behavior: not merging lists
            push(Value{V: res})
        case "-":
            b, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            a, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            push(Value{V: a.V - b.V})
        case "*":
            b, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            a, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            push(Value{V: a.V * b.V})
        case "/":
            b, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            a, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if b.V == 0 { return Value{}, ErrNodeRightValInvalid }
            push(Value{V: a.V / b.V})
        case "^":
            b, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            a, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if a.V == 0 && b.V == 0 { return Value{}, ErrNodeLeftValInvalid }
            if b.V < 0 { return Value{}, ErrNodeRightValInvalid }
            res := 1
            for i := 0; i < b.V; i++ { res *= a.V }
            push(Value{V: res})
        case "d":
            sidesV, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            timesV, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            sides := sidesV.V
            times := timesV.V
            if times <= 0 || times > 10000 { return Value{}, ErrNodeLeftValInvalid }
            if sides <= 0 || sides > 10000 { return Value{}, ErrNodeRightValInvalid }
            rolls := make([]int, 0, times)
            sum := 0
            for i := 0; i < times; i++ {
                rnum := r.rng.Intn(sides) + 1
                rolls = append(rolls, rnum)
                sum += rnum
            }
            push(Value{V: sum, Meta: rolls, MetaEnable: true})
        case "k": // keep highest k
            param, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            left, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if !left.MetaEnable {
                // nothing to keep, treat as normal
                push(left)
                continue
            }
            k := param.V
            if k <= 0 { return Value{}, ErrNodeRightValInvalid }
            rolls := append([]int(nil), left.Meta...)
            // sort descending
            for i := 0; i < len(rolls)-1; i++ {
                for j := i+1; j < len(rolls); j++ {
                    if rolls[j] > rolls[i] { rolls[i], rolls[j] = rolls[j], rolls[i] }
                }
            }
            if k > len(rolls) { k = len(rolls) }
            sel := rolls[:k]
            s := 0
            for _, vv := range sel { s += vv }
            push(Value{V: s, Meta: sel, MetaEnable: true})
        case "a":
            // additive chain: roll `times` groups on m-sided dice; any roll >= threshold adds to next round
            rightV, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftV, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            times := leftV.V
            threshold := rightV.V
            if times < 0 || times > 10000 { return Value{}, ErrNodeLeftValInvalid }
            if threshold <= 0 || threshold > 10000 { return Value{}, ErrNodeRightValInvalid }
            m := 10 // default sides for 'a'
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
                    // count success as >= threshold
                    if rnum >= threshold {
                        total++
                    }
                }
                // guard to avoid runaway loops
                if len(meta) > 10000 { break }
            }
            push(Value{V: total, Meta: meta, MetaEnable: len(meta) > 0})
        case "c":
            // condensed chain: roll groups and sum max of each round; continue while any roll >= threshold
            rightC, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftC, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            timesC := leftC.V
            thresholdC := rightC.V
            if timesC < 0 || timesC > 10000 { return Value{}, ErrNodeLeftValInvalid }
            if thresholdC <= 0 || thresholdC > 10000 { return Value{}, ErrNodeRightValInvalid }
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
                    if rnum > maxv { maxv = rnum }
                    if rnum >= thresholdC { nextC++ }
                }
                totalC += maxv
                if len(metaC) > 10000 { break }
            }
            push(Value{V: totalC, Meta: metaC, MetaEnable: len(metaC) > 0})
        case "b":
            // bonus mechanic (percentile-like) based on Python implementation
            // right times param determines how many bonus digits to roll
            paramB, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftB, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            // leftB unused in Python for b's logic except validation; we'll follow bounds
            if paramB.V < 0 { return Value{}, ErrNodeRightValInvalid }
            if paramB.V > 10000 { return Value{}, ErrNodeRightValInvalid }
            if leftB.V > 10000 { return Value{}, ErrNodeLeftValInvalid }
            base := r.rng.Intn(100) + 1
            units := base % 10
            tens := base / 10
            minRoll := 10
            var rolls []int
            for i := 0; i < paramB.V; i++ {
                rr := r.rng.Intn(10) // 0..9
                // python uses random(1,10)-1 with special handling
                if units == 0 && rr == 0 {
                    rr = 10
                }
                rolls = append(rolls, rr)
                if rr < minRoll { minRoll = rr }
            }
            var out int
            if units == 0 && minRoll == 0 {
                out = 100
            } else if tens > minRoll {
                out = units + minRoll*10
            } else {
                out = base
            }
            push(Value{V: out, Meta: rolls, MetaEnable: len(rolls) > 0})
        case "p":
            // punish mechanic (mirror of b)
            paramP, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftP, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if paramP.V < 0 { return Value{}, ErrNodeRightValInvalid }
            if paramP.V > 10000 { return Value{}, ErrNodeRightValInvalid }
            if leftP.V > 10000 { return Value{}, ErrNodeLeftValInvalid }
            baseP := r.rng.Intn(100) + 1
            unitsP := baseP % 10
            tensP := baseP / 10
            maxRoll := -1
            var rollsP []int
            for i := 0; i < paramP.V; i++ {
                rr := r.rng.Intn(10)
                if unitsP == 0 && rr == 0 {
                    rr = 10
                }
                rollsP = append(rollsP, rr)
                if rr > maxRoll { maxRoll = rr }
            }
            var outP int
            if unitsP == 0 && maxRoll == 0 {
                outP = 100
            } else if tensP < maxRoll {
                outP = unitsP + maxRoll*10
            } else {
                outP = baseP
            }
            push(Value{V: outP, Meta: rollsP, MetaEnable: len(rollsP) > 0})
        case "=":
            // assignment: pop right value then left placeholder
            rightA, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftA, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if !leftA.IsTemp {
                return Value{}, ErrNodeLeftValInvalid
            }
            // set temp
            if r.temp == nil { r.temp = map[int]int{} }
            r.temp[leftA.TempIndex] = rightA.V
            // also sync into ValueTable as T{n} (uppercase key) for cross-RD visibility
            if r.ValueTable == nil { r.ValueTable = map[string]int{} }
            tkey := strings.ToUpper(fmt.Sprintf("t%d", leftA.TempIndex))
            r.ValueTable[tkey] = rightA.V
            push(Value{V: rightA.V})
        case "lp":
            // repeat/loop operator: left meta list repeated right times
            paramLp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftLp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            timesLp := paramLp.V
            if timesLp <= 0 { return Value{}, ErrNodeRightValInvalid }
            // if left has string templates, produce repeated strings with {i} replaced by 1-based index
            if leftLp.MetaStr != nil && len(leftLp.MetaStr) > 0 {
                // flatten left templates into a single sequence
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
            rollsLp := leftLp.Meta
            if !leftLp.MetaEnable {
                rollsLp = []int{leftLp.V}
            }
            newList := make([]int, 0, len(rollsLp)*timesLp)
            for i := 0; i < timesLp; i++ {
                newList = append(newList, rollsLp...)
            }
            sumLp := 0
            for _, vv := range newList { sumLp += vv }
            push(Value{V: sumLp, Meta: newList, MetaEnable: len(newList) > 0})
        case "q": // keep lowest q
            param, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            left, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if !left.MetaEnable {
                push(left)
                continue
            }
            q := param.V
            if q <= 0 { return Value{}, ErrNodeRightValInvalid }
            rolls := append([]int(nil), left.Meta...)
            // sort ascending
            for i := 0; i < len(rolls)-1; i++ {
                for j := i+1; j < len(rolls); j++ {
                    if rolls[j] < rolls[i] { rolls[i], rolls[j] = rolls[j], rolls[i] }
                }
            }
            if q > len(rolls) { q = len(rolls) }
            sel := rolls[:q]
            s := 0
            for _, vv := range sel { s += vv }
            push(Value{V: s, Meta: sel, MetaEnable: true})
        case "kh", "kl", "dh", "dl":
            // pop param then left
            paramOp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftOp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            n := paramOp.V
            if n <= 0 { return Value{}, ErrNodeRightValInvalid }
            // prepare list of pairs [value, raw]
            rollsRaw := leftOp.Meta
            if !leftOp.MetaEnable {
                rollsRaw = []int{leftOp.V}
            }
            // build slice of pairs
            type pair struct { v int; raw int }
            pairs := make([]pair, 0, len(rollsRaw))
            for _, rv := range rollsRaw {
                pairs = append(pairs, pair{v: rv, raw: rv})
            }
            if len(pairs) == 0 { return Value{}, ErrNodeLeftValInvalid }
            // sort according to operator
            // simple bubble sort for small sizes
            if tok == "kh" || tok == "dh" {
                // sort desc
                for i := 0; i < len(pairs)-1; i++ {
                    for j := i+1; j < len(pairs); j++ {
                        if pairs[j].v > pairs[i].v { pairs[i], pairs[j] = pairs[j], pairs[i] }
                    }
                }
            } else {
                // kl or dl: sort asc
                for i := 0; i < len(pairs)-1; i++ {
                    for j := i+1; j < len(pairs); j++ {
                        if pairs[j].v < pairs[i].v { pairs[i], pairs[j] = pairs[j], pairs[i] }
                    }
                }
            }
            var newPairs []pair
            if tok == "kh" || tok == "kl" {
                if n <= len(pairs) {
                    newPairs = pairs[:n]
                } else {
                    newPairs = pairs
                }
            } else {
                // dh / dl: drop first n
                if n < len(pairs) {
                    newPairs = pairs[n:]
                } else {
                    newPairs = pairs
                }
            }
            sel := make([]int, len(newPairs))
            sum := 0
            for i, p := range newPairs { sel[i] = p.v; sum += p.v }
            push(Value{V: sum, Meta: sel, MetaEnable: true})
        case "min", "max":
            paramOp2, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftOp2, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            n2 := paramOp2.V
            if n2 <= 0 { return Value{}, ErrNodeRightValInvalid }
            rollsRaw2 := leftOp2.Meta
            if !leftOp2.MetaEnable {
                rollsRaw2 = []int{leftOp2.V}
            }
            resList := make([]int, len(rollsRaw2))
            sum2 := 0
            for i, rv := range rollsRaw2 {
                if tok == "max" {
                    if rv > n2 { rv = n2 }
                } else {
                    if rv < n2 { rv = n2 }
                }
                resList[i] = rv
                sum2 += rv
            }
            push(Value{V: sum2, Meta: resList, MetaEnable: true})
        case "f":
            // fudge/fate dice: left times roll in [-1,1], sum
            rightF, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftF, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            if rightF.V <= 1 || rightF.V > 10000 { return Value{}, ErrNodeRightValInvalid }
            if leftF.V <= 0 || leftF.V > 10000 { return Value{}, ErrNodeLeftValInvalid }
            rollsF := make([]int, 0, leftF.V)
            sumF := 0
            for i := 0; i < leftF.V; i++ {
                // random(-1,1)
                rnum := r.rng.Intn(3) - 1
                rollsF = append(rollsF, rnum)
                sumF += rnum
            }
            push(Value{V: sumF, Meta: rollsF, MetaEnable: true})
        case "sp":
            // select position: pop param then left; returns single element at position
            paramSp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftSp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            idx := paramSp.V
            rollsSp := leftSp.Meta
            if !leftSp.MetaEnable {
                // single value: only index 1 or -1 valid
                if idx == 1 || idx == -1 {
                    val := leftSp.V
                    push(Value{V: val, Meta: []int{val}, MetaEnable: true})
                    continue
                }
                return Value{}, ErrNodeLeftValInvalid
            }
            if idx == 0 { return Value{}, ErrNodeRightValInvalid }
            var pos int
            if idx > 0 { pos = idx - 1 } else { pos = len(rollsSp) + idx }
            if pos < 0 || pos >= len(rollsSp) { return Value{}, ErrNodeRightValInvalid }
            v := rollsSp[pos]
            push(Value{V: v, Meta: []int{v}, MetaEnable: true})
        case "tp":
            // take position: pop param then left; removes element at position and returns sum of remaining
            paramTp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            leftTp, ok := pop()
            if !ok { return Value{}, ErrNodeStackEmpty }
            idx2 := paramTp.V
            rollsTp := leftTp.Meta
            if !leftTp.MetaEnable {
                // single value: removing it yields empty -> sum 0
                if idx2 == 1 || idx2 == -1 {
                    push(Value{V: 0, Meta: []int{}, MetaEnable: false})
                    continue
                }
                return Value{}, ErrNodeLeftValInvalid
            }
            if idx2 == 0 { return Value{}, ErrNodeRightValInvalid }
            var pos2 int
            if idx2 > 0 { pos2 = idx2 - 1 } else { pos2 = len(rollsTp) + idx2 }
            if pos2 < 0 || pos2 >= len(rollsTp) { return Value{}, ErrNodeRightValInvalid }
            newList := append([]int{}, rollsTp[:pos2]...) 
            if pos2+1 < len(rollsTp) { newList = append(newList, rollsTp[pos2+1:]...) }
            sumTp := 0
            for _, vv := range newList { sumTp += vv }
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

// evalTokens evaluates a token slice and supports short-circuit ternary ?: by
// locating top-level '?' and matching ':'; non-ternary slices are evaluated
// by converting to RPN and using evalRPN.
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
            if tokens[i] != ")" { continue }
            // find matching open for this close
            d := 0
            oi := -1
            for k := i; k >= 0; k-- {
                if tokens[k] == ")" { d++ }
                if tokens[k] == "(" { d-- }
                if d == 0 { oi = k; break }
            }
            if oi == -1 { return Value{}, ErrUnknownGenerate }
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
        if derr != "" { return Value{}, derr }
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
                if tokens[i] == "(" { d++ }
                if tokens[i] == ")" { d-- }
                if d == 0 { match = i; break }
            }
            if match == len(tokens)-1 {
                // outer parentheses wrap entire expression: strip and continue
                tokens = tokens[1:len(tokens)-1]
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
                        if derr != "" { return Value{}, derr }
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
    rpn, err := toRPN(tokens)
    if err != nil {
        return Value{}, ErrUnknownGenerate
    }
    return r.evalRPN(rpn)
}
