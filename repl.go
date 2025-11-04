package gonedice

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RunREPL 启动一个交互式的REPL(读取-求值-打印循环)环境，用于解析和执行OneDice表达式
// 该函数从标准输入读取用户输入的表达式，计算结果并输出到标准输出
// 主要用于命令行界面，让用户能够交互式地测试和使用gonedice库的功能
//
// 使用说明:
//   - 输入OneDice表达式(如"1d6+2")进行掷骰计算
//   - 输入"quit"或"exit"退出REPL
//   - 输入空行会继续等待下一个输入
//
// 输出结果包含:
//   - Value: 计算结果的数值
//   - Meta: 详细的掷骰过程信息
//   - Detail: 格式化的结果详情
func RunREPL() {
	fmt.Println("gonedice REPL - 输入 OneDice 表达式或 'quit' 退出")

	// 历史记录数组
	var history []string

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" {
			break
		}

		// 添加到历史记录，但避免重复添加
		if len(line) > 0 {
			if len(history) == 0 || history[len(history)-1] != line {
				history = append(history, line)
			}
		}

		r := New(line, nil)
		r.Roll()
		res := r.Result()
		if res.Error != "" {
			fmt.Println("Error:", res.Error)
			continue
		}
		fmt.Printf("Value: %d\n", res.Value)
		fmt.Printf("Meta: %v\n", res.MetaTuple)
		fmt.Printf("Detail: %s\n", res.Detail)
	}

	// 简单提示如何查看历史
	if len(history) > 0 {
		fmt.Println("\n本次会话的输入历史:")
		for i, cmd := range history {
			fmt.Printf("%3d: %s\n", i+1, cmd)
		}
	}
}
