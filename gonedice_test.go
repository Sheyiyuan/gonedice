package gonedice

import (
	"math/rand"
	"testing"
)

func TestArithmetic(t *testing.T) {
	r := New("1+2*3", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if res.Value != 7 {
		t.Fatalf("expected 7 got %d", res.Value)
	}
}

func TestDiceFixedSeed(t *testing.T) {
	r := New("2d6k1", nil)
	// set deterministic rng
	r.rng = rand.New(rand.NewSource(42))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if res.Value < 1 || res.Value > 6 {
		t.Fatalf("dice k1 out of range: %d", res.Value)
	}
}

func TestVarReplace(t *testing.T) {
	vt := map[string]int{"STR": 5}
	r := New("{STR}+2", vt)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if res.Value != 7 {
		t.Fatalf("expected 7 got %d", res.Value)
	}
}

func TestBAndP(t *testing.T) {
	r := New("1b3", nil)
	r.rng = rand.New(rand.NewSource(123))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error b: %v", res.Error)
	}
	if res.Value < 1 || res.Value > 100 {
		t.Fatalf("b out of range: %d", res.Value)
	}

	r2 := New("1p3", nil)
	r2.rng = rand.New(rand.NewSource(456))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error p: %v", res2.Error)
	}
	if res2.Value < 1 || res2.Value > 100 {
		t.Fatalf("p out of range: %d", res2.Value)
	}
}

func TestAandC(t *testing.T) {
	r := New("3a5", nil)
	r.rng = rand.New(rand.NewSource(777))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error a: %v", res.Error)
	}
	if res.Value < 0 {
		t.Fatalf("a negative: %d", res.Value)
	}

	// custom faces parameter m: 3a5m6 should roll faces in 1..6
	r3 := New("3a5m6", nil)
	r3.rng = rand.New(rand.NewSource(777))
	r3.Roll()
	res3 := r3.Result()
	if res3.Error != "" {
		t.Fatalf("unexpected error a with m: %v", res3.Error)
	}
	for _, v := range res3.MetaTuple {
		vi := v.(int)
		if vi < 1 || vi > 6 {
			t.Fatalf("a with m produced out-of-range die %d", vi)
		}
	}

	r2 := New("3c5", nil)
	r2.rng = rand.New(rand.NewSource(888))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error c: %v", res2.Error)
	}
	if res2.Value < 0 {
		t.Fatalf("c negative: %d", res2.Value)
	}

	// custom faces parameter for c
	r4 := New("3c5m6", nil)
	r4.rng = rand.New(rand.NewSource(888))
	r4.Roll()
	res4 := r4.Result()
	if res4.Error != "" {
		t.Fatalf("unexpected error c with m: %v", res4.Error)
	}
	for _, v := range res4.MetaTuple {
		vi := v.(int)
		if vi < 1 || vi > 6 {
			t.Fatalf("c with m produced out-of-range die %d", vi)
		}
	}
}

func TestKH_KL_DH_DL(t *testing.T) {
	r := New("4d6kh3", nil)
	r.rng = rand.New(rand.NewSource(42))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error kh: %v", res.Error)
	}
	if len(res.MetaTuple) != 3 {
		t.Fatalf("kh meta length expected 3 got %d", len(res.MetaTuple))
	}
	sum := 0
	for _, v := range res.MetaTuple {
		vi := v.(int)
		sum += vi
	}
	if sum != res.Value {
		t.Fatalf("kh sum mismatch: meta sum %d vs value %d", sum, res.Value)
	}

	r2 := New("4d6kl3", nil)
	r2.rng = rand.New(rand.NewSource(43))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error kl: %v", res2.Error)
	}
	if len(res2.MetaTuple) != 3 {
		t.Fatalf("kl meta length expected 3 got %d", len(res2.MetaTuple))
	}
	sum2 := 0
	for _, v := range res2.MetaTuple {
		vi := v.(int)
		sum2 += vi
	}
	if sum2 != res2.Value {
		t.Fatalf("kl sum mismatch: meta sum %d vs value %d", sum2, res2.Value)
	}

	r3 := New("4d6dh1", nil)
	r3.rng = rand.New(rand.NewSource(44))
	r3.Roll()
	res3 := r3.Result()
	if res3.Error != "" {
		t.Fatalf("unexpected error dh: %v", res3.Error)
	}
	if len(res3.MetaTuple) != 3 {
		t.Fatalf("dh meta length expected 3 got %d", len(res3.MetaTuple))
	}

	r4 := New("4d6dl1", nil)
	r4.rng = rand.New(rand.NewSource(45))
	r4.Roll()
	res4 := r4.Result()
	if res4.Error != "" {
		t.Fatalf("unexpected error dl: %v", res4.Error)
	}
	if len(res4.MetaTuple) != 3 {
		t.Fatalf("dl meta length expected 3 got %d", len(res4.MetaTuple))
	}
}

func TestKH_KL_EdgeCases(t *testing.T) {
	// n greater than roll count -> should select all available
	r := New("2d6kh3", nil)
	r.rng = rand.New(rand.NewSource(1))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error kh n>len: %v", res.Error)
	}
	if len(res.MetaTuple) != 2 {
		t.Fatalf("kh n>len meta length expected 2 got %d", len(res.MetaTuple))
	}

	// duplicate values: ensure selection handles duplicates correctly
	r2 := New("4d1kh2", nil) // all rolls are 1
	r2.rng = rand.New(rand.NewSource(2))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error kh duplicates: %v", res2.Error)
	}
	if len(res2.MetaTuple) != 2 {
		t.Fatalf("kh duplicates meta length expected 2 got %d", len(res2.MetaTuple))
	}
	sum := 0
	for _, v := range res2.MetaTuple {
		sum += v.(int)
	}
	if sum != res2.Value {
		t.Fatalf("kh duplicates sum mismatch: meta sum %d vs value %d", sum, res2.Value)
	}

	// non-meta left (single value) should be treated as single-element list
	r3 := New("5kh3", nil)
	r3.Roll()
	res3 := r3.Result()
	if res3.Error != "" {
		t.Fatalf("unexpected error kh single-left: %v", res3.Error)
	}
	if len(res3.MetaTuple) != 1 {
		t.Fatalf("kh single-left meta length expected 1 got %d", len(res3.MetaTuple))
	}
	if res3.Value != 5 {
		t.Fatalf("kh single-left expected value 5 got %d", res3.Value)
	}
}

func TestDH_DL_EdgeCases(t *testing.T) {
	// dh/dl dropping more than length results in empty meta
	r := New("2d6dh3", nil)
	r.rng = rand.New(rand.NewSource(3))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error dh n>=len: %v", res.Error)
	}
	if len(res.MetaTuple) != 0 {
		t.Fatalf("dh n>=len expected empty meta got %v", res.MetaTuple)
	}

	r2 := New("2d6dl3", nil)
	r2.rng = rand.New(rand.NewSource(4))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error dl n>=len: %v", res2.Error)
	}
	if len(res2.MetaTuple) != 0 {
		t.Fatalf("dl n>=len expected empty meta got %v", res2.MetaTuple)
	}
}

func TestTupleAndKhIntegration(t *testing.T) {
	// tuple literal should be treated as multi-element operand for kh
	r := New("[4,2,6]kh2", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error tuple kh: %v", res.Error)
	}
	if len(res.MetaTuple) != 2 {
		t.Fatalf("tuple kh meta len expected 2 got %d", len(res.MetaTuple))
	}
	sum := 0
	for _, v := range res.MetaTuple {
		sum += v.(int)
	}
	if sum != res.Value {
		t.Fatalf("tuple kh sum mismatch: meta sum %d vs value %d", sum, res.Value)
	}
}

func TestTuplePolymorphismD(t *testing.T) {
	seed := int64(12345)
	r1 := New("[2,3]d6", nil)
	r1.rng = rand.New(rand.NewSource(seed))
	r1.Roll()
	res1 := r1.Result()
	if res1.Error != "" {
		t.Fatalf("unexpected error tuple d: %v", res1.Error)
	}

	r2 := New("3d6", nil)
	r2.rng = rand.New(rand.NewSource(seed))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error explicit d: %v", res2.Error)
	}

	if res1.Value != res2.Value {
		t.Fatalf("tuple d polymorphism mismatch: %d vs %d", res1.Value, res2.Value)
	}
}

func TestTupleElementsWithExpressionsKh(t *testing.T) {
	// one element is a dice expression that always yields 1
	r := New("[1d1,2]kh1", nil)
	r.rng = rand.New(rand.NewSource(77))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error tuple expr kh: %v", res.Error)
	}
	if res.Value != 2 {
		t.Fatalf("expected kh to pick 2 got %d", res.Value)
	}
	if len(res.MetaTuple) != 1 {
		t.Fatalf("expected single meta element got %v", res.MetaTuple)
	}
}

func TestTupleSpTpBasic(t *testing.T) {
	r := New("[3,4,5]sp2", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error tuple sp: %v", res.Error)
	}
	if res.Value != 4 {
		t.Fatalf("tuple sp expected 4 got %d", res.Value)
	}
	r2 := New("[3,4,5]tp2", nil)
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error tuple tp: %v", res2.Error)
	}
	// tp removes element at pos 2, remaining sum should be 3+5=8
	if res2.Value != 8 {
		t.Fatalf("tuple tp expected 8 got %d", res2.Value)
	}
}

func TestBDeterministic(t *testing.T) {
	seed := int64(424242)
	param := 3
	r := New("1b3", nil)
	r.rng = rand.New(rand.NewSource(seed))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error b deterministic: %v", res.Error)
	}
	// replicate RNG sequence to compute expected result
	rng := rand.New(rand.NewSource(seed))
	tens := rng.Intn(10)
	units := rng.Intn(10)
	extras := make([]int, param)
	for i := 0; i < param; i++ {
		extras[i] = rng.Intn(10)
	}
	// for bonus, tens is replaced by max(extras)
	mx := extras[0]
	for _, v := range extras[1:] {
		if v > mx {
			mx = v
		}
	}
	tens = mx
	var expected int
	if tens == 0 && units == 0 {
		expected = 100
	} else {
		expected = tens*10 + units
	}
	if res.Value != expected {
		t.Fatalf("b deterministic mismatch: expected %d got %d", expected, res.Value)
	}
	if len(res.MetaTuple) != param+2 {
		t.Fatalf("b deterministic meta len expected %d got %d", param+2, len(res.MetaTuple))
	}
	// first two are tens and units, extras follow
	for i := 0; i < param; i++ {
		vi := res.MetaTuple[i+2].(int)
		if vi != extras[i] {
			t.Fatalf("b deterministic meta element %d expected %d got %d", i, extras[i], vi)
		}
	}
}

func TestPDeterministic(t *testing.T) {
	seed := int64(424243)
	param := 4
	r := New("1p4", nil)
	r.rng = rand.New(rand.NewSource(seed))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error p deterministic: %v", res.Error)
	}
	rng := rand.New(rand.NewSource(seed))
	tens := rng.Intn(10)
	units := rng.Intn(10)
	extras := make([]int, param)
	for i := 0; i < param; i++ {
		extras[i] = rng.Intn(10)
	}
	// for punish, tens is replaced by min(extras)
	mn := extras[0]
	for _, v := range extras[1:] {
		if v < mn {
			mn = v
		}
	}
	tens = mn
	var expected int
	if tens == 0 && units == 0 {
		expected = 100
	} else {
		expected = tens*10 + units
	}
	if res.Value != expected {
		t.Fatalf("p deterministic mismatch: expected %d got %d", expected, res.Value)
	}
	if len(res.MetaTuple) != param+2 {
		t.Fatalf("p deterministic meta len expected %d got %d", param+2, len(res.MetaTuple))
	}
	for i := 0; i < param; i++ {
		vi := res.MetaTuple[i+2].(int)
		if vi != extras[i] {
			t.Fatalf("p deterministic meta element %d expected %d got %d", i, extras[i], vi)
		}
	}
}

func TestBZeroHundredAndParamZero(t *testing.T) {
	// find a seed that yields tens==0 && units==0 for b
	found := int64(0)
	for s := int64(1); s <= 10000; s++ {
		rng := rand.New(rand.NewSource(s))
		if rng.Intn(10) == 0 && rng.Intn(10) == 0 {
			found = s
			break
		}
	}
	if found == 0 {
		t.Fatalf("could not find seed producing tens==0 && units==0 in range")
	}
	// now run b with that seed and param>0 to ensure it becomes 100
	r := New("1b2", nil)
	r.rng = rand.New(rand.NewSource(found))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error b zero hundred: %v", res.Error)
	}
	if res.Value != 100 {
		t.Fatalf("expected b to yield 100 for tens==0&&units==0 seed %d got %d", found, res.Value)
	}
	// test param=0: no extras rolled, just tens/units
	r2 := New("1b0", nil)
	seed := int64(31415)
	r2.rng = rand.New(rand.NewSource(seed))
	r2.Roll()
	res2 := r2.Result()
	// compute expected
	rng := rand.New(rand.NewSource(seed))
	tens := rng.Intn(10)
	units := rng.Intn(10)
	var expected int
	if tens == 0 && units == 0 {
		expected = 100
	} else {
		expected = tens*10 + units
	}
	if res2.Value != expected {
		t.Fatalf("b param=0 expected %d got %d", expected, res2.Value)
	}
}

func TestPZeroHundredAndParamZero(t *testing.T) {
	found := int64(0)
	for s := int64(1); s <= 10000; s++ {
		rng := rand.New(rand.NewSource(s))
		if rng.Intn(10) == 0 && rng.Intn(10) == 0 {
			found = s
			break
		}
	}
	if found == 0 {
		t.Fatalf("could not find seed producing tens==0 && units==0 in range for p")
	}
	r := New("1p2", nil)
	r.rng = rand.New(rand.NewSource(found))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error p zero hundred: %v", res.Error)
	}
	if res.Value != 100 {
		t.Fatalf("expected p to yield 100 for tens==0&&units==0 seed %d got %d", found, res.Value)
	}
	// param=0
	seed := int64(271828)
	r2 := New("1p0", nil)
	r2.rng = rand.New(rand.NewSource(seed))
	r2.Roll()
	res2 := r2.Result()
	rng := rand.New(rand.NewSource(seed))
	tens := rng.Intn(10)
	units := rng.Intn(10)
	var expected int
	if tens == 0 && units == 0 {
		expected = 100
	} else {
		expected = tens*10 + units
	}
	if res2.Value != expected {
		t.Fatalf("p param=0 expected %d got %d", expected, res2.Value)
	}
}

func TestMinMax(t *testing.T) {
	r := New("3d10max5", nil)
	r.rng = rand.New(rand.NewSource(99))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error max: %v", res.Error)
	}
	// each meta element must be <= 5
	for _, v := range res.MetaTuple {
		vi := v.(int)
		if vi > 5 {
			t.Fatalf("max failed, element %d > 5", vi)
		}
	}

	r2 := New("3d10min5", nil)
	r2.rng = rand.New(rand.NewSource(100))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error min: %v", res2.Error)
	}
	// each meta element must be >= 5
	for _, v := range res2.MetaTuple {
		vi := v.(int)
		if vi < 5 {
			t.Fatalf("min failed, element %d < 5", vi)
		}
	}
}

func TestSpTp(t *testing.T) {
	r := New("4d6sp2", nil)
	r.rng = rand.New(rand.NewSource(2025))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error sp: %v", res.Error)
	}
	if len(res.MetaTuple) != 1 {
		t.Fatalf("sp meta length expected 1 got %d", len(res.MetaTuple))
	}

	r2 := New("4d6tp2", nil)
	r2.rng = rand.New(rand.NewSource(2026))
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error tp: %v", res2.Error)
	}
	if len(res2.MetaTuple) != 3 {
		t.Fatalf("tp meta length expected 3 got %d", len(res2.MetaTuple))
	}
	sum := 0
	for _, v := range res2.MetaTuple {
		vi := v.(int)
		sum += vi
	}
	if sum != res2.Value {
		t.Fatalf("tp sum mismatch: meta sum %d vs value %d", sum, res2.Value)
	}
}

func TestTernaryAndTempAndLp(t *testing.T) {
	r := New("0?2:3", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error ternary: %v", res.Error)
	}
	if res.Value != 3 {
		t.Fatalf("ternary failed expected 3 got %d", res.Value)
	}

	r2 := New("1?2:3", nil)
	r2.Roll()
	res2 := r2.Result()
	if res2.Error != "" {
		t.Fatalf("unexpected error ternary true: %v", res2.Error)
	}
	if res2.Value != 2 {
		t.Fatalf("ternary true failed expected 2 got %d", res2.Value)
	}

	r3 := New("$t=7+$t", nil)
	r3.Roll()
	res3 := r3.Result()
	if res3.Error != "" {
		t.Fatalf("unexpected error temp assign: %v", res3.Error)
	}
	if res3.Value != 14 {
		t.Fatalf("temp assign expected 14 got %d", res3.Value)
	}

	r4 := New("3d6lp2", nil)
	r4.rng = rand.New(rand.NewSource(111))
	r4.Roll()
	res4 := r4.Result()
	if res4.Error != "" {
		t.Fatalf("unexpected error lp: %v", res4.Error)
	}
	// meta length should be 6
	if len(res4.MetaTuple) != 6 {
		t.Fatalf("lp meta len expected 6 got %d", len(res4.MetaTuple))
	}
}

func TestFOperator(t *testing.T) {
	r := New("5f3", nil)
	r.rng = rand.New(rand.NewSource(2027))
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error f: %v", res.Error)
	}
	if len(res.MetaTuple) != 5 {
		t.Fatalf("f meta length expected 5 got %d", len(res.MetaTuple))
	}
	sum := 0
	for _, v := range res.MetaTuple {
		vi := v.(int)
		sum += vi
	}
	if sum != res.Value {
		t.Fatalf("f sum mismatch: meta sum %d vs value %d", sum, res.Value)
	}
}

func TestTernaryShortCircuitTrue(t *testing.T) {
	r := New("(1?($t1=5):($t1=6))+$t1", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error ternary true: %v", res.Error)
	}
	if res.Value != 10 {
		t.Fatalf("ternary true expected 10 got %d", res.Value)
	}
	if r.temp[1] != 5 {
		t.Fatalf("temp write from true branch expected 5 got %d", r.temp[1])
	}
}

func TestTernaryShortCircuitFalse(t *testing.T) {
	r := New("(0?($t1=5):($t1=6))+$t1", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error ternary false: %v", res.Error)
	}
	if res.Value != 12 {
		t.Fatalf("ternary false expected 12 got %d", res.Value)
	}
	if r.temp[1] != 6 {
		t.Fatalf("temp write from false branch expected 6 got %d", r.temp[1])
	}
}

func TestTernaryShortCircuitAvoidsError(t *testing.T) {
	r := New("1?1:(1/0)", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error shortcircuit avoid: %v", res.Error)
	}
	if res.Value != 1 {
		t.Fatalf("shortcircuit avoid expected 1 got %d", res.Value)
	}
}

func TestLpStringTemplateSimple(t *testing.T) {
	r := New("\"{i}\"lp3", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error lp string simple: %v", res.Error)
	}
	if len(res.MetaTuple) != 3 {
		t.Fatalf("lp string simple meta len expected 3 got %d", len(res.MetaTuple))
	}
	if res.MetaTuple[0].(string) != "1" || res.MetaTuple[2].(string) != "3" {
		t.Fatalf("lp string simple content mismatch: %v", res.MetaTuple)
	}
}

func TestLpStringTemplateComplex(t *testing.T) {
	r := New("\"x{i}y\"lp2", nil)
	r.Roll()
	res := r.Result()
	if res.Error != "" {
		t.Fatalf("unexpected error lp string complex: %v", res.Error)
	}
	if len(res.MetaTuple) != 2 {
		t.Fatalf("lp string complex meta len expected 2 got %d", len(res.MetaTuple))
	}
	if res.MetaTuple[0].(string) != "x1y" || res.MetaTuple[1].(string) != "x2y" {
		t.Fatalf("lp string complex content mismatch: %v", res.MetaTuple)
	}
}
