## 自适应限流算法



#### 1) 一般限流

```js
一般我们会选择 `漏斗桶/令牌桶` 算法来进行限流, 确实能够保护系统不被拖垮。其`核心思想`有两点:
1) 设置指标, 固定一个漏斗或者固定发送令牌的速度
2) 超过指标限制流量进入

根据这两个特点, 我们很容易推出会遇到什么`问题`:
1) 指标不好定, 设置流量的阈值是什么?
2) 当突然出现流量高峰的时候, 是需要人工介入去调整的

总结就是传统限流比较被动, 不能够自适应流量的变化
```



#### 2) 自适应限流

```js
对于自适应限流来说, 一般都是结合系统的 `Load`、`CPU` 使用率以及应用的入口 `QPS`、`平均响应时间`和`并发量`等几个维度的监控指标，通过自适应的流控策略, 让系统的入口流量和系统的负载达到一个平衡，让系统尽可能跑在最大吞吐量的同时保证系统整体的稳定性
```



#### 3) 实现

我们参考`kratos` 和 `go-zero` , 来看一下自适应限流具体是如何实现的



##### 1) 基本公式

```js
# 1) 计算单点cpu, 得出一个 [0~1000]的数字表示 0~100%的cpu
cpu = ( 周期内用户使用 / 周期内系统总共使用 ) * 1e3

# 2) 滑动窗口cpu计算 (指数加权平均算法)
// 一般decay=0.95, 表示衰退率
// t表示时间周期, t-1 表示上一个时间周期
windowCpu =  cpu = cpuᵗ⁻¹ * decay + cpuᵗ * (1 - decay)

# 3) 计算是否应该丢弃
1) cpu 大于预定值, 比如900
2) 周期内请求数超过允许的最大请求数,计算方式如下
// winBucketPerSec: 每秒内的采样数量,
// 计算方式:
// int64(time.Second)/(int64(conf.Window)/int64(conf.WinBucket)),
// conf.Window默认值10s, conf.WinBucket默认值100.
// 简化下公式: 1/(10/100) = 10, 所以每秒内的采样数就是10
// maxQPS = maxPass * winBucketPerSec
// minRT = min average response time in milliseconds
// maxQPS * minRT / milliseconds_per_second
maxFlight = b.maxPass()*b.minRT()*b.winBucketPerSec)/1e3
```



##### 2) 计算cpu

此处只计算`linux`下的cpu, 根据 `cgroup`计算

文件路径:  `internal/cpu/cgroup.go`

```js
# 1) cgroup文件地址, 读取相关信息
/proc/{pid}/cgroup

// 得到类似如下信息 (我这里读的是某个docker进程的数据)
11:cpuset:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
10:memory:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
9:devices:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
8:blkio:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
7:hugetlb:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
6:perf_event:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
5:freezer:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
4:net_cls,net_prio:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
3:pids:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
2:cpu,cpuacct:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30
1:name=systemd:/docker/290247cde1fff59d5322068be83a7c7629f4454ac0960a89e6856ea041970b30

# 2) /sys/fs/cgroup 再把对应cpu拼上cgroup根路径读取对应信息

# 3) 最后计算出cpu的使用率
```



##### 3) 计算滑动窗口cpu

算法:  指数加权平均算法 ( `moving average `)

时间周期: time.Millisecond * 500, 有`1s`的冷却时间

公式 `cpu = cpuᵗ⁻¹ * decay + cpuᵗ * (1 - decay)`

窗口: `10s`的窗口内划分`100个bucket`, 衰退率 `decay=0.95`

文件路径: `internal/middleware/bbr.go:CpuProc()`

```js
// CpuProc update cpu in every 250 Millisecond
func CpuProc() {
	ticker := time.NewTicker(time.Millisecond * 250)
	defer func() {
		ticker.Stop()
		if err := recover(); err != nil {
			fmt.Println("cpuProc fail, e:", err)
			go CpuProc()
		}
	}()

	for range ticker.C {
		stat := &internal.Stat{}
		internal.LoadStat(stat)
		preCpu := atomic.LoadInt64(&cpu)

		// cpu = cpuᵗ⁻¹ * decay + cpuᵗ * (1 - decay)
		curCpu := int64(float64(preCpu)*decay + float64(stat.Usage)*(1.0-decay))

		atomic.StoreInt64(&cpu, curCpu)
		fmt.Printf("old-self-cpu: %v, now-self-cpu:%v \n", preCpu, curCpu)
	}

}
```



##### 4) 计算窗口内允许的最大请求数

公式: `maxFlight = b.maxPass()*b.minRT()*b.winBucketPerSec)/1e3`

文件:`internal/middleware/bbr.go::maxFlight()`

实际上就是每个bucket内最大的请求通过数和最小的响应时间相乘, 即为`maxFlight`

如果cpu大于预设值或者请求数大于`maxFlight`, 则判定为需要丢掉请求



###### 1) maxPass

```js
// maxPass 单个采样窗口在一个采样周期中的最大的请求数,
// 默认的采样窗口是10s, 采样bucket数量100
func (b *BBR) maxPass() int64 {
	maxPassCache := b.maxPassCache.Load()
	if maxPassCache != nil {
		ps := maxPassCache.(*CounterCache)
		if b.timespan(ps.time) < 1 {
			return ps.val
		}
	}

	rawMaxPass := int64(b.passStat.Reduce(func(iterator metric.Iterator) float64 {
		var result = 1.0
		for i := 1; iterator.Next() && i < b.conf.WinBucket; i++ {
			bucket := iterator.Bucket()
			count := 0.0
			for _, point := range bucket.Points {
				count += point
			}
			result = math.Max(result, count)
		}
		return result
	}))

	if rawMaxPass == 0 {
		rawMaxPass = 1
	}

	b.maxPassCache.Store(&CounterCache{
		val:  rawMaxPass,
		time: time.Now(),
	})

	return rawMaxPass
}
```



###### 2) minRT

```js
// minRT 单个采样窗口中最小的响应时间
func (b *BBR) minRT() int64 {
	minRtCache := b.minRtCache.Load()
	if minRtCache != nil {
		rc := minRtCache.(*CounterCache)
		if b.timespan(rc.time) < 1 {
			return rc.val
		}
	}

	rawMinRt := int64(math.Ceil(b.rtStat.Reduce(func(iterator metric.Iterator) float64 {
		var res = math.MaxFloat64

		for i := 1; iterator.Next() && i < b.conf.WinBucket; i++ {
			bucket := iterator.Bucket()
			if len(bucket.Points) == 0 {
				continue
			}

			total := 0.0
			for _, point := range bucket.Points {
				total += point
			}
			avg := total / float64(bucket.Count)
			res = math.Min(res, avg)
		}

		return res

	})))

	if rawMinRt <= 0 {
		rawMinRt = 1
	}

	b.minRtCache.Store(&CounterCache{
		val:  rawMinRt,
		time: time.Now(),
	})

	return rawMinRt
}
```



###### 3) maxFlight

```js
// current window max flight
func (b *BBR) maxFlight() int64 {
	// winBucketPerSec: 每秒内的采样数量,
	// 计算方式:
	// int64(time.Second)/(int64(conf.Window)/int64(conf.WinBucket)),
	// conf.Window默认值10s, conf.WinBucket默认值100.
	// 简化下公式: 1/(10/100) = 10, 所以每秒内的采样数就是10
	// maxQPS = maxPass * winBucketPerSec
	// minRT = min average response time in milliseconds
	// maxQPS * minRT / milliseconds_per_second
	return int64(
		math.Floor(
			float64(
				b.maxPass()*b.minRT()*b.winBucketPerSec)/1e3 + 0.5,
		),
	)

}
```



###### 4) shouldDrop

```js
// Cooling time: 1s
func (b *BBR) shouldDrop() bool {
	// not overload
	if b.cpu() < b.conf.CPUThreshold {
		preDropTime, _ := b.preDrop.Load().(time.Duration)
		// didn't drop before
		if preDropTime == 0 {
			return false
		}

		// in cooling time duration, 1s
		// should not drop
		if time.Since(initTime)-preDropTime <= time.Second {
			inFlight := atomic.LoadInt64(&b.inFlight)
			return inFlight > 1 && inFlight > b.maxFlight()
		}

		// store this drop time as pre drop time
		b.preDrop.Store(time.Duration(0))
		return false
	}

	// overload case
	inFlight := atomic.LoadInt64(&b.inFlight)
	shouldDrop := inFlight > 1 && inFlight > b.maxFlight()

	if shouldDrop {
		preDropTime, _ := b.preDrop.Load().(time.Duration)
		if preDropTime != 0 {
			return shouldDrop
		}
		b.preDrop.Store(time.Since(initTime))
	}

	return shouldDrop
}
```



##### 5) rollingCounter

窗口统计, 核心数据结构为:

```js
// Bucket contains multiple float64 points.
// 环形链表
type Bucket struct {
	Points []float64 // all of the points
	Count  int64     // this bucket point length
	next   *Bucket
}

// Window contains multiple buckets.
Window struct {
	buckets []Bucket
	size    int
}
```



#### 4) 源码地址

`https://github.com/sado0823/go-bbr-ratelimit`



#### 5) 参考资料

* [alibaba-sentinel](https://github.com/alibaba/sentinel-golang/wiki/%E7%B3%BB%E7%BB%9F%E8%87%AA%E9%80%82%E5%BA%94%E6%B5%81%E6%8E%A7)
* [kratos-bbr](https://github.com/go-kratos/kratos)
* [go-zero-shedding](https://github.com/tal-tech/go-zero)
* [EMA algorithm](https://blog.csdn.net/m0_38106113/article/details/81542863)
* [cgroup-/proc/stat](https://man7.org/linux/man-pages/man5/proc.5.html)
* [cgroup-cpu](https://man7.org/linux/man-pages/man7/cgroups.7.html)

