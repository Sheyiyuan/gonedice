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
