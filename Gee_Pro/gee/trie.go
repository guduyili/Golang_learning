package gee

import (
	"fmt"
	"strings"
)

type node struct {
	pattern  string  // 待匹配路由，例如 /p/:lang
	part     string  // 路由中的一部分，例如 :lang
	children []*node // 子节点，例如 [doc, tutorial, intro]
	isWild   bool    // 是否精确匹配，part 含有 : 或 * 时为 true
}

func (n *node) String() string {
	return fmt.Sprintf("node{pattern=%s, part=%s, isWild=%t}", n.pattern, n.part, n.isWild)
}

// 第一个匹配成功的节点 用于插入 (精确或通配符)
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 所有匹配成功的节点 用于查找（用于深度搜素）
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

// 插入节点
func (n *node) insert(pattern string, parts []string, height int) {
	// 递归处理每一层pattern
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	// 处理当前层
	part := parts[height]
	// 查找是否存在可复用的子节点
	child := n.matchChild(part)
	if child == nil {
		// 若不存在， 创建新节点并标记是否为通配符
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	// 递归处理下一层
	child.insert(pattern, parts, height+1)
}

// 查找匹配节点
func (n *node) search(parts []string, height int) *node {
	// 走到路径末尾 或 遇到通配符节点
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n // 找到匹配节点
	}
	// 处理当前层
	part := parts[height]
	children := n.matchChildren(part)
	for _, child := range children {
		// 深度优先搜索 找到匹配的即返回
		ret := child.search(parts, height+1)
		if ret != nil {
			return ret
		}
	}
	return nil
}

// travel 遍历节点，存储匹配的节点到列表中
func (n *node) travel(list *[]*node) {
	//将当前节点的有效路由加入列表
	if n.pattern != "" {
		*list = append(*list, n)
	}

	//深度遍历子节点
	for _, child := range n.children {
		child.travel(list)
	}
}
