# Go 语言学习笔记

> C++ 工程师转 Go 的关键概念对照

---

## 结构体 vs 类

Go 没有 class，用**结构体 + 方法**代替：

```go
// 定义结构体（≈ C++ class 的成员变量）
type Person struct {
    Name string
    Age  int
}

// 方法（指针接收者 ≈ this 指针）
func (p *Person) GetName() string {
    return p.Name
}
```

**C++ 对比**：
```cpp
class Person {
public:
    std::string getName() const { return name; }
private:
    std::string name;
};
```

---

## 没有构造函数，用 New/Get 函数代替

```go
// Go 惯用模式
func NewPerson(name string, age int) *Person {
    return &Person{Name: name, Age: age}
}

// 单例模式
func GetInstance() *Config {
    once.Do(func() {
        instance = &Config{}
    })
    return instance
}
```

**C++ 对比**：Go 的 `NewXxx()` 就是 C++ 构造函数的手动版本。

---

## 全局变量（包级变量）

```go
// 包级别 var = C++ 的全局变量
var GlobalCount int           // 值为 0（零值）
var GlobalMap = make(map[string]string)  // 需要 make 初始化

func init() {
    // init() 函数在包被 import 时自动执行，比 main() 更早
    GlobalCount = 100
}
```

**注意**：`var m map[string]string` 声明后值是 `nil`，直接写入会 panic，必须用 `make()` 初始化。

---

## sync.Once — 线程安全单例

```go
var (
    instance *Config
    once     sync.Once
)

func GetConfig() *Config {
    once.Do(func() {
        instance = &Config{}
    })
    return instance
}
```

**C++ 对比**：`std::call_once` + `std::once_flag`

---

## sync.Mutex / sync.RWMutex

```go
// 互斥锁（读写都排他）
mu sync.Mutex
mu.Lock()
defer mu.Unlock()   // defer ≈ C++ 的 RAII / 析构自动释放

// 读写锁（读并发，写排他，适合读多写少）
mu sync.RWMutex
mu.RLock()          // 读锁
defer mu.RUnlock()
mu.Lock()           // 写锁
defer mu.Unlock()
```

**C++ 对比**：`std::mutex` / `std::shared_mutex`（C++17）

---

## defer — 自动资源释放

```go
func readFile(path string) {
    f, _ := os.Open(path)
    defer f.Close()   // 函数返回时自动执行，无论是否出错
    // ... 使用 f
}
```

**C++ 对比**：`std::lock_guard`、`std::unique_ptr` 的 RAII 机制。Go 的 `defer` 更通用——任何清理逻辑都可以 defer。

---

## goroutine + channel ≈ std::async + std::future

```go
ch := make(chan Result, 1)

go func() {
    result := doWork()      // 异步执行
    ch <- result            // 发送结果到 channel
}()

r := <-ch                   // 阻塞等待结果
```

**C++ 对比**：
```cpp
auto future = std::async(std::launch::launch, []{ return doWork(); });
auto result = future.get();
```

---

## interface — 隐式实现

Go 的接口是**隐式实现**的，不需要显式声明 `implements`：

```go
type Writer interface {
    Write([]byte) (int, error)
}

// 只要实现了 Write 方法，就自动满足 Writer 接口
type MyWriter struct{}
func (w *MyWriter) Write(data []byte) (int, error) { ... }
```

**C++ 对比**：Go interface ≈ C++ 纯虚类，但不需要继承声明。

---

## 错误处理 — 返回值而非异常

```go
result, err := doSomething()
if err != nil {
    // 处理错误
    return err
}
```

**C++ 对比**：Go 没有 try/catch，`error` 是普通返回值。类似 C 的返回值检查模式。

---

## nil 对比

| Go 的 nil | C++ 的对应 |
|---|---|
| nil pointer | `nullptr` |
| nil map | 未初始化的 `std::map` |
| nil slice | 空 `std::vector`（但 len=0） |
| nil interface | 无值的 `std::unique_ptr` |

---

## 常用语法速查

```go
// 循环
for i := 0; i < 10; i++ { }          // 经典 for
for _, v := range slice { }           // 遍历切片
for k, v := range myMap { }           // 遍历 map

// 条件（不需要括号）
if err != nil { }
switch v := x.(type) {                // 类型断言（≈ dynamic_cast）
    case int: ...
    case string: ...
}

// 切片操作
s := []int{1, 2, 3}                   // 字面量初始化
s = append(s, 4)                      // 追加
```

---

## 相关笔记

- [[项目架构总览]]
- [[emsxmldb模块]]
- [[技术要点]]
