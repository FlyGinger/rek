# rek

`rek`是一个典型的造轮子项目，并且还造的不圆。

## 使用方法

``` go
r := Compile("(a*|b*)[0-9]?[a-zA-Z]+(x?y?z?|abc)")
if r.Match(input[i]) {
    // ...
}
```

首先使用`Compile`处理正则表达式，得到一个`rek`数据结构，然后就可以使用该结构的`Match`方法获知输入字符串与正则表达式是否匹配（可以多次调用`Match`方法）。以下是正则表达式支持的功能。

- 字面量：`a`（支持Unicode）。
- 通配符：`.`。`.`等价于`[^\n]`。
- （否定）字符类：`[a-z123]`、`[^a-z123]`。（否定）字符类中`a-z`形式的表示中，`a`必须小于等于`z`。`-`放在首个（在否定字符类中是除`^`之外的首个）或最后一个字符的位置会被当作字面量处理，同理`^`不放在第一个也会被当作字面量。字符类中元字符（除了`^`和`-`）不使用逃逸符就可以表示该字符本身，例如`[.]`、`[\.]`和`\.`等价。正则表达式中多余的`]`会被当作字面量，例如`a]`是合法的；但是多余的`[`会引发错误，例如`a[`。
- 重复：`*`、`+`、`?`。
- 逃逸符：`\`。逃逸符后只能放`\`、`(`、`)`、`*`、`+`、`?`、`|`、`.`、`[`、`]`、`t`、`r`、`n`。在（否定）字符类中，逃逸符之后还可以放`^`和`-`。
- 括号：`()`。正则表达式中多余的`)`会被当作字面量，例如`a)`是合法的；但是多余的`(`会引发错误，例如`a(`。

## 基准测试

``` plaintext
goos: windows
goarch: amd64
pkg: rek
cpu: AMD Ryzen 5 3500U with Radeon Vega Mobile Gfx  
BenchmarkCompileMatch
BenchmarkCompileMatch-8         13426     90205 ns/op
BenchmarkMatch
BenchmarkMatch-8              4893344     220.8 ns/op
BenchmarkRE2CompileMatch
BenchmarkRE2CompileMatch-8      97099     11854 ns/op
BenchmarkRE2Match
BenchmarkRE2Match-8            773724      1894 ns/op
PASS
```

如果是先`Compile`然后`Match`，那么`rek`比`re2`慢得多。如果把`Compile`过程排除在外，`rek`快一些。但是`rek`实现的功能少多了，当然应该快一些。

## 实现

以下实现`rek`的主要步骤。

1. （`Compile`）将正则表达式转换为NFA，然后将NFA转化为DFA；
2. （`Match`）根据得到的DFA判断输入字符串是否匹配正则表达式。

### 从正则表达式到NFA

首先是正则语言的基本特性，字面量`a`、连接`ab`、选择`a|b`、Kleene闭包`a*`以及派生的`a+`和`a?`。其次，单纯的字面量`a`肯定是不够的，还需要通配符`.`、字符类`[]`和否定字符类`[^]`来描述某个范围。逃逸符`\`也是必要的，用于表示某些空白符或元字符本身，例如`\n`、`\*`等。最后，还需要`()`来调节优先级，括号的引入将大大提升实现的复杂性。这样数数之后，要实现的功能就不少了。

- 字面量、通配符及（否定）字符类：`a`、`.`、`[a-z123]`、`[^a-z123]`。
- 重复：`*`、`+`、`?`。
- 逃逸符：`\`。
- 括号：`()`。

正则表达式的语法（指`rek`的语法）并不复杂，可以借助一个栈，在一次线性扫描的时间复杂度内完成语法过程。

#### 与正则表达式对应的NFA

字面量`a`是一个正则表达式，那么它对应的NFA应该是这样的：

``` plaintext
       a
>(0) -----> ((1))
```

其中`>(x)`代表起始状态，`((x))`代表终结状态。给定两个正则表达式`R`和`S`，它们分别转换成了两个NFA，那么对正则表达式`RS`、`R|S`、`R*`、`R+`和`R?`对应的NFA是什么样子呢？

##### `R*`、`R+`和`R?`

重复的实现非常简单。对于`R*`，添加从起始状态到终结状态和从终结状态到起始状态的两条无条件转移；对于`R+`，添加从终结状态到起始状态的无条件转移；对于`R?`，添加从起始状态到终结状态的无条件转移。以下分别是`R*`、`R+`和`R?`对应的NFA。

``` plaintext
      e                                   e
    -----                               -----
   |     |                             |     |
   |     v                             |     v
>(0) ... ((1))    >(0) ... ((1))    >(0) ... ((1))
   ^     |           ^     |
   |  e  |           |  e  |
    -----             -----
```

##### `RS`

最简单、最安全的平凡方法是这样的：我们直接将`R`的终结状态和`S`的起始状态用一个无条件转移连接起来。

``` plaintext
            R    e     S
... -----> (x) -----> (y) -----> ...
```

但是这种平凡方法有一个问题，以`abc`这个正则表达式举例，以下分别是使用平凡方法构造出的NFA和实际上最简化的NFA。

``` plaintext
       a          e          b          e          c
>(0) -----> (1) -----> (2) -----> (3) -----> (4) -----> ((5))

       a          b          c
>(0) -----> (1) -----> (2) -----> ((3))
```

设`R`是由`n`个字面量连接而成的正则表达式，使用平凡方法构造的NFA的状态数量是`2n`，转移数量是`2n-1`。而最简形式的NFA只有`n+1`个状态和`n`个转移，当`n`足够大时，使用平凡方法构造的NFA的状态数量和转移数量都将是最简形式的2倍。实现最简形式方法也很简单：不再是添加无条件转移，而是直接将`R`的终结状态和`S`的起始状态合并。

``` plaintext
         R      S
... -----> (xy) -----> ...
```

这样一来状态和转移数量就大幅度减少了。然而新的问题又出现了，对于不符合上述要求的正则表达式（仅由字面量连接而成）例如`a*b*`，如果直接使用合并状态的方法，那么构造出来的NFA是错误的。

``` plaintext
       a          b
>(0) -----> (1) -----> ((2))
   ^        | ^        |
   |   e    | |   e    |
    --------   --------
```

这个NFA现在可以接受`abab`这样的字符串，该NFA与`a*b*`不等价了。因此最终的解决方案是：

- 如果`R`没有从终结状态出发的转移或者`S`没有到达起始状态的转移，那么将`R`的终结状态和`S`的起始状态合并；
- 否则，使用一条无条件转移将`R`的终结状态和`S`的起始状态连接起来。

##### `R|S`

首先还是介绍简单安全的平凡方法：添加两个状态分别作为新NFA的起始状态和终结状态，然后添加四条无条件转移，连接两个新状态和`R`与`S`。

``` plaintext
       e         R        e
>(0) -----> (x) ... (y) -----> ((1))
   |                           ^
   |   e         S        e    |
    ------> (z) ... (w) -------
```

显然，使用这种平凡方法合成的NFA会具有更多的状态数量。以正则表达式`a|b`举例，以下分别是平凡方法生成的NFA和最简形式的NFA。

``` plaintext
       e          a          e
>(0) -----> (1) -----> (2) -----> ((5))
   |                              ^
   |   e          b          e    |
    ------> (3) -----> (4) -------

       a
>(0) -----> ((1))
   |        ^
   |   b    |
    --------
```

设`R`是`n`个字面量选择而成的正则表达式，使用平凡方法构造的NFA的状态数量是`4n-2`，转移数量是`5n-4`。而最简形式的NFA只有`2`个状态和`n`个转移，当`n`足够大时，使用平凡方法构造的NFA的状态数量是最简形式的`2n-1`倍，转移数量是最简形式的5倍。实现最简形式方法也很简单：不再是添加无条件转移，而是直接将`R`的起始状态和`S`的起始状态合并，将`R`的终结状态和`S`的终结状态合并。

``` plaintext
      R
     ...
   |     |
   |     v
>(0)     ((1))
   |     ^
   |  S  |
     ...
```

对于不符合上述要求的正则表达式（仅由字面量选择而成），例如`a+|b+`，如果直接使用合并状态的方法，那么构造出来的NFA是错误的。

``` plaintext
        a
    --------
   |        |
   |    e   v
>(0) <----- ((1))
   |        ^
   |    b   |
    --------
```

这个NFA现在可以接受`abab`这样的字符串了，该NFA与`a+|b+`不等价了。因此最终的解决方案是：

- 对于起始状态：
  - 如果`R`和`S`中都没有到达起始状态的转移，那么合并两个起始状态作为新NFA的起始状态。
  - 如果`R`中有到达起始状态的转移，而`S`中没有，那么添加一条从`S`起始状态到达`R`起始状态的无条件转移，并将`S`的起始状态作为新NFA的起始状态。反之类似。
  - 如果`R`和`S`中都有到达起始状态的转移，那么添加一个新状态作为新NFA的起始状态，并添加从该新状态分别到达`R`和`S`起始状态的无条件转移。
- 对于终结状态：
  - 如果`R`和`S`中都没有从终结状态出发的转移，那么合并两个终结状态，作为新NFA的终结状态。
  - 如果`R`中有从终结状态出发的转移，而`S`中没有，那么添加一条从`R`终结状态到达`S`终结状态的无条件转移，并将`S`的终结状态作为新`NFA`的终结状态。反之类似。
  - 如果`R`和`S`中都有从终结状态出发的转移，那么添加一个新状态作为新NFA的终结状态，并添加从`R`和`S`到达该新状态的无条件转移。

#### 数据结构

如何表示转移条件呢？使用两个数组`lower`和`upper`，当输入的字符`c`能够找到满足`lower[i] <= c <= upper[i]`的`i`时，表示`c`符合该转移条件。另外，无条件转移使用一个额外的标志来表示。

- `a`：`lower:[97]`，`upper:[97]`
- `.`：`lower:[0,11]`，`upper:[9,1114111]`。
- `[a-z]`：`lower:[97]`，`upper:[122]`。
- `[^a-z]`：`lower:[0,123]`，`upper:[96,1114111]`。

``` go
type nfaTransfer struct {
    target *nfaState
    empty  bool
    lower  []rune
    upper  []rune
}

type nfaState struct {
    transfers []*nfaTransfer
}

type nfa struct {
    toStart []*nfaTransfer
    toEnd   []*nfaTransfer
    states  []*nfaState
}
```

而存储NFA的数据结构就简单多了。其中，`states`是一个二维数据，每个`nfaState`中存储了从该状态出发的转移。此外，由于针对NFA的各种处理中，均会保证第一个（下标是0）状态是起始状态，最后一个状态是终结状态，因此无需在数据结构中使用额外的标志。NFA的连接和合并的算法中，如果需要合并两个NFA的起始或者终止状态，那么就需要修改所有到达起始或终止状态的转移。因此数据结构中维护了`toStart`和`toEnd`两个数组，以避免遍历NFA的所有状态来寻找需要修改的转移。

#### `group`

前文提到，算法借助一个栈来实现正则表达式到NFA的转移。`group`是这个算法要用到的一个小函数，它可以将栈中最后一个`(`（如果有）之后的所有元素按要求拼接起来。假设栈中只有三种元素，NFA、`(`标志和`|`标志。

1. 将栈中元素弹出，直到遇到栈空、`(`标志或`|`标志。
2. 将弹出的所有NFA按正确顺序连接在一起。
3. 如果步骤1中遇到的是`|`标志，则将步骤2的连接结果放入一个临时栈，然后回到步骤1。
4. 将临时栈中的所有NFA选择在一起，然后将结果放回栈中。

总而言之，`group`将栈中最后一层`()`中的所有NFA拼接起来，然后放回栈中。

#### 算法流程

1. 从正则表达式中读取一个字符，记为`ch`。
2. 如果`ch`是`(`，向栈中压入一个`(`标志。
3. 如果`ch`是`)`，执行`group`函数。
4. 如果`ch`是`*`、`+`或`?`，则对栈顶的NFA执行对应的操作。
5. 如果`ch`是`|`，向栈中压入一个`|`标志。
6. 否则，`ch`是字面量、通配符或者（否定）字符类，应该向栈中压入一个新的NFA。（如果`ch`是`[`，代表一个（否定）字符类的开始，还需要继续从正则表达式中读取后续字符来获得该（否定）字符类的具体细节）。

上述算法中还省略了一些检查。例如，在步骤3、步骤4和步骤5中，栈必须不空且栈顶元素必须是一个NFA而不能是标志。而且整个正则表达式中，括号必须是匹配的。可以使用一个额外变量`parCnt`来检查括号匹配：当遇到`(`时，`parCnt`加1；当遇到`)`时，`parCnt`减1。如果遇到`)`时`parCnt`已经为0，或者算法结束时`parCnt`不为0，那么整个正则表达式中，括号是不匹配的。

### 从NFA到DFA

NFA是非确定性有限自动机，输入的字符可能同时满足多个转移的条件，而它总能选择事后看来正确的选项。这种非确定性导致NFA的运行过程不是一个算法，我们需要先将其转换为DFA（或者使用其他方法模拟NFA运行）。

1. 设`S`是NFA中状态的集合。算法开始时，`S`只有NFA的起始状态，即`{0}`。
2. 对于`S`中的每个状态`p`，计算从`p`出发不消耗任何输入所能到达的所有状态构成的集合，记为`E(p)`。从`p`出发不消耗任何输入所能到达的状态包括`p`本身和从`p`出发只通过无条件转移所能到达的状态。
3. 令`S`等于所有`E(p)`的并集。
4. 对于字符表中的每个字符`a`，计算从`S`中的任意状态出发只消耗且必须消耗恰好一个`a`所能到达的所有状态的集合`Sa`。
5. 对上一步得到的每一个状态集合`Sa`，回到步骤2，令`S=Sa`，重复执行本算法，直到不再生成新的状态集合。

对于Unicode字符表来说，对每个状态集合遍历字符表所需要的时间是不可接受的。好在我们使用范围来表示转移条件，一般来说这个范围不会十分复杂，按范围遍历可以大幅减少计算量。

另一个问题是，一个具有`n`个状态的NFA，可以列出多少种状态集合？如果不算空集，那么一共是`(2^n)-1`个。显然，上面NFA到DFA的转换算法的最坏时间复杂度是指数的。实际中使用的正则表达式一般不会导致最坏的时间复杂度，先写出来看看性能怎么样吧。

#### 求解`E(p)`

这里使用Floyd-Warshall算法，能够非常方便地求出所有`p`的`E(p)`。首先，构造二维`bool`数组`can`，`can[i][j]`表示`pj`是否在`E(pi)`中。然后，遍历NFA中的所有转移，对于从`pi`出发，到达`pj`的无条件转移，令`can[i][j]`为`true`。

Floyd-Warshall算法的思想是，对于任意的三个节点`i`、`j`、`k`，是否有一条从`i`出发经过`k`到达`j`的路径。因此算法的最后一步是一个三层循环（设`N`是NFA中状态的数量）：对于`0 <= k < N`，`0 <= i < N`，`0 <= j < N`，`can[i][j] = can[i][j] || can[i][k] && can[k][j]`。花费`O(N^3)`的时间复杂度，可以求出NFA中所有状态的`E(p)`集合。

#### 求解下一步

对于一个NFA状态集合`S`，需要计算出从`S`出发，消耗一个字符能够到达哪些状态。首先，我们收集从`S`中任意状态出发的所有转移。然后，我们从中取出任意两个，合并为一个。重复这一过程，直到只剩一个转移。这就是算法的基本流程。对于取出的任意两个转移，它们可能是这样的：

``` plaintext
transfer 0: lower[0, 11], upper[9, 1114111] -> state 5
transfer 1: lower[120], upper[120] -> state 8
```

将其合并后，应该如下所示。Unicode的范围`[0,1114111]`将被划分为数个区间，每个区间对应一个目标集合。对应区间的目标集合为空的将会被省略。

``` plaintext
lower    upper    target set
[  0,       9] -> state 5
[  9,     119] -> state 5
[120,     120] -> state 5, state 8
[121, 1114111] -> state 5
```

#### DFA状态表示

DFA的每个状态都是NFA的状态集合。当算法得到一个新的DFA状态`{2,5,8}`时，如何判断该状态是否出现过？此时需要一个哈希表来记录已经出现过的状态，但是有两个问题：第一，如何处理乱序和重复，即`{5,2,8}`、`{2,5,8,2}`与`{2,5,8}`是相同的状态；第二，Go语言的`map`类型并不支持`slice`作为索引。

对于第一个问题，可以通过去重和排序来解决。首先将状态集合进行排序，然后再线性扫描中去掉重复的元素。这可以在`O(NlnN)`的时间复杂度内解决。然而在词法分析器中，描述`token`的正则表达式一般都比较简短，形成的NFA也就状态数不多。因此可以使用一个`bool`数组来表示DFA状态，数组长度等于NFA状态数量，使用`true`和`false`表示某NFA状态是否包含在该DFA状态中。例如，一个共有11个状态的NFA，DFA状态`{2,5,8}`应该表示为`[]bool{false, false, true, false, false, true, false, false, true, false, false, false}`。这个方法耗费`O(1)`的时间复杂度，但是代价是`O(N)`的空间复杂度。我使用了第二种方法，毕竟它编程起来也比较简单。

第二个问题，可以通过自定义哈希函数的办法来解决。实现方法也非常简单，首先取一小一大两个质数`p`、`m`，例如`p = 3`，`m = 100000007`。然后进行如下计算：

``` go
hash := 0
for i := range set {
    hash *= p
    if set[i] {
       hash += 1
    }
    hash %= m
}
```

这相当于把DFA的状态当作一个`p`进制的数字，将其十进制值对`m`取模后，就得到了哈希值。使用该哈希值作为键，就可以将它放在`map`中了。当然，两个哈希值相等的DFA状态并不一定相同，还是需要线性扫描一遍才能确认的。

#### 数据结构

存储DFA的数据结构与NFA类似，但是由于DFA的确定性，还是有点不同的。

``` go
type dfaTransfer struct {
    target       int
    lower, upper rune
}

type dfaState struct {
    isEnd     bool
    transfers []dfaTransfer
}

type dfa struct {
    states []dfaState
}
```

为了方便模拟DFA运行，`dfaTransfer`的`lower`和`upper`变成了`rune`而不是`rune`数组。同时，DFA性质和转换的具体算法决定了同一个`dfaState`中的`transfers`数组中的所有`dfaTransfer`是按照`lower`和`upper`升序排列的，并且它们之间不会有重叠。这样以来就可以使用二分法，在`O(lnN)`的时间复杂度中确定是否存在符合条件的转移。

#### 算法流程

1. 对于NFA的所有状态`p`，求出`E(p)`。
2. 使用广度优先搜索的算法，建立`queue`，并将NFA起始状态对应的`E(0)`放入队列中。同时，将`E(0)`作为DFA的起始状态。
3. 从队列中取出一个状态集合`S`，求解下一步，得到各个区间和对应的目标集合。
4. 对于每个状态集合，求集合中所有状态`p`对应的`E(p)`的并集`T`。在DFA中添加从`S`到`T`的转移。如果`T`是一个新DFA状态，将其添加到`queue`中。
5. 如果`queue`不空，回到步骤3。

### 模拟DFA运行

模拟DFA运行的过程就非常简单了，初始状态设置为0，然后不断根据输入字符更新状态，直到找不到下一个状态（返回`false`），或者输入结束（返回是否停留在终结状态）。