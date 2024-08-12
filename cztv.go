package main

import (
	"fmt"
	"github.com/grafov/m3u8"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

// ================ 网络格式转换 =====================

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

// FormatBytes 将字节数转换为自适应单位（B, KB, MB, GB, TB）
func FormatBytes(bytes uint64) string {
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2fTB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// ================ tsURl储存结构体 =====================

// FixedSizeSlice 固定大小的切片，当容量满了就删除最先添加的元素，且不允许重复元素
type FixedSizeSlice struct {
	size  int
	slice []string
}

// NewFixedSizeSlice 创建一个新的 FixedSizeSlice
func NewFixedSizeSlice(size int) *FixedSizeSlice {
	return &FixedSizeSlice{
		size:  size,
		slice: make([]string, 0, size),
	}
}

// Add 添加一个元素到切片中，如果切片已满，则删除最先添加的元素
// 如果元素已经存在，返回 false；否则添加元素并返回 true
func (f *FixedSizeSlice) Add(value string) bool {
	// 判断元素已存在
	if !(len(f.slice) == 0) {
		for i := len(f.slice) - 1; i >= 0; i-- {
			if value == f.slice[i] {
				return false // 元素已存在
			}
		}
	}

	// 如果切片已满，则删除最先添加的元素
	if len(f.slice) >= f.size {
		f.slice = f.slice[1:]
	}

	// 添加新元素
	f.slice = append(f.slice, value)
	return true
}

// GetSlice 返回当前的切片
func (f *FixedSizeSlice) GetSlice() []string {
	return f.slice
}

// =====================================

// 下载m3u8文件
func downloadM3U8(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// 解析m3u8文件 返回ts列表
func parseM3U8(content string) (*m3u8.MediaPlaylist, error) {
	reader := strings.NewReader(content)
	p, listType, err := m3u8.DecodeFrom(reader, true)
	if err != nil {
		return nil, err
	}

	if listType != m3u8.MEDIA {
		return nil, fmt.Errorf("not a media playlist")
	}

	return p.(*m3u8.MediaPlaylist), nil
}

// 下载ts文件
func downloadSegment(url string, counter *int64) error {

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 下载读取后直接丢弃
	bytesRead, err := io.Copy(io.Discard, resp.Body)

	atomic.AddInt64(counter, bytesRead)

	fmt.Printf("%s == %s == %s \n", FormatBytes(uint64(atomic.LoadInt64(counter))), FormatBytes(uint64(bytesRead)), url)

	return err
}

func main() {

	error := make(chan int, 1)

	var counter int64

	fun := func() {

		// 捕获 panic
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("程序运行异常:", r)
				error <- 1
				return
			}
		}()

		// 直播
		//m3u8Arr := []string{
		//	//"http://l.cztvcloud.com/channels/lantian/SXyunhe1/720p.m3u8",
		//	//"http://live.hznet.tv:1935/live/live2/500K/tzwj_video.m3u8",
		//	//"http://live.hznet.tv:1935/live/live3/500K/tzwj_video.m3u8",
		//	//"http://live.hznet.tv:1935/live/live1/500k/tzwj_video.m3u8",
		//	//"http://lives.jnnews.tv/video/s10001-JNTV3/index.m3u8",
		//	//"http://lives.jnnews.tv/video/s10001-JNTV-1/index.m3u8",
		//	//"http://stream.zztvzd.com/3/sd/live.m3u8?shandd",
		//	//"http://stream.zztvzd.com/2/sd/live.m3u8",
		//	//"http://lmt.scqstv.com/live1/live1.m3u8",
		//	//"http://stream.qhbtv.com/qhds/sd/live.m3u8",
		//	//"http://stream.thmz.com/wxtv4/sd/live.m3u8",
		//	//"http://live.dxhmt.cn:9081/tv/11326-1.m3u8",
		//	//"http://tvpull.dxhmt.cn:9081/tv/11322-1.m3u8",
		//	//"http://tvpull.dxhmt.cn:9081/tv/10181-1.m3u8",
		//	//"http://ali-m-l.cztv.com/channels/lantian/channel001/1080p.m3u8",
		//	//"http://ali-m-l.cztv.com/channels/lantian/channel002/1080p.m3u8",
		//	//"http://ali-m-l.cztv.com/channels/lantian/channel003/1080p.m3u8",
		//	//"http://ali-m-l.cztv.com/channels/lantian/channel004/1080p.m3u8",
		//	//"http://ali-m-l.cztv.com/channels/lantian/channel006/1080p.m3u8",
		//	//"http://ali-m-l.cztv.com/channels/lantian/channel007/1080p.m3u8",
		//	//"http://ali-m-l.cztv.com/channels/lantian/channel008/1080p.m3u8",
		//	//"http://liveflash.sxrtv.com/live/sxwshd.m3u8?sub_m3u8=true&edge_slice=true",
		//	//"http://l.cztvcloud.com/channels/lantian/SXwuyi1/720p.m3u8",
		//	//"http://l.cztvcloud.com/channels/lantian/SXpinghu1/720p.m3u8",
		//}

		// 音频
		// 超稳定源 https://medium.com/@k21_79139/%E4%B8%AD%E5%9C%8B%E9%9B%BB%E5%8F%B0%E7%9B%B4%E6%92%AD%E6%BA%90%E5%88%97%E8%A1%A8-61733d389509
		m3u8Arr := []string{
			"http://sk.cri.cn/hyhq.m3u8",
			"http://sk.cri.cn/hxfh.m3u8",
			"http://sk.cri.cn/nhzs.m3u8",
			"http://sk.cri.cn/am846.m3u8",
			"http://sk.cri.cn/905.m3u8",
			"http://sk.cri.cn/915.m3u8",
			"http://ngcdn001.cnr.cn/live/zgzs/index.m3u8",
			"http://ngcdn002.cnr.cn/live/jjzs/index.m3u8",
			"http://ngcdn003.cnr.cn/live/yyzs/index.m3u8",
			"http://ngcdn004.cnr.cn/live/dszs/index.m3u8",
			"http://ngcdn005.cnr.cn/live/zhzs/index.m3u8",
			"https://brtv-radiolive.rbc.cn/alive/fm945.m3u8",
			//"https://brtv-radiolive.rbc.cn/alive/fm1073.m3u8",
			//"https://brtv-radiolive.rbc.cn/alive/am603.m3u8",
			//"https://brtv-radiolive.rbc.cn/alive/fm1006.m3u8",
			"http://stream3.hndt.com/now/4pcovD2L/chunklist.m3u8",
			//"http://stream3.hndt.com/now/MdOpB4zP/chunklist.m3u8",
			//"http://stream3.hndt.com/now/SXJtR4M4/chunklist.m3u8",
			//"http://stream3.hndt.com/now/8bplFuwp/chunklist.m3u8",
			//"http://stream3.hndt.com/now/PHucVOu2/chunklist.m3u8",
		}

		// 随机提取一个
		randomIndex := rand.Intn(len(m3u8Arr))
		m3u8URL := m3u8Arr[randomIndex]

		// 创建储存容器 不需要重复下载ts文件，因为m3u8是5秒更新一次
		// 缓冲区为20个ts文件
		fixedSlice := NewFixedSizeSlice(20)

		var sleepSeconds int64 = 5

		for true {
			// 循环获取m3u8文件

			base, err := url.Parse(m3u8URL)
			if err != nil {
				fmt.Println("解析取m3u8URl错误：", err.Error())
				error <- 1
				return
			}

			// 访问url地址获取m3u8文件内容
			content, err := downloadM3U8(m3u8URL)
			if err != nil {
				fmt.Println("获取m3u8文件错误：", err.Error())
				error <- 1
				return
			}

			// 有的M3u8文件不规范 #EXTINF:8.00 后面不带,号，所以这里需要正则表达式替换 避免解析报错
			// 定义正则表达式
			re := regexp.MustCompile(`#EXTINF:.*\n`)
			// 在 M3U8 数据中查找所有匹配项
			matches := re.FindAllStringSubmatch(content, -1)
			for _, match := range matches {
				if len(match) == 1 {
					if !strings.Contains(match[0], ",") {
						content = re.ReplaceAllString(content, "#EXTINF:8.00,\n")
						break
					}
				}
			}

			// 解析m3u8获取ts列表
			mediaPlaylist, err := parseM3U8(content)
			if err != nil {
				fmt.Println("解析m3u8错误：", err.Error())
				error <- 1
				return
			}

			// 遍历ts列表
			for _, segment := range mediaPlaylist.Segments {
				if segment != nil {
					// 解析 TS 文件的完整 URL
					tsURL, err := url.Parse(segment.URI)
					if err != nil {
						fmt.Println("解析取m3u8URl错误：", err.Error())
						continue
					}
					fullURL := base.ResolveReference(tsURL).String()

					// 不重复才下载（只下载未下载过的ts文件）
					if fixedSlice.Add(fullURL) {
						// 下载ts文件
						err = downloadSegment(fullURL, &counter)
						sleepSeconds = int64(segment.Duration * 1000)
						if err != nil {
							continue
						}
					}
				}
			}
			time.Sleep(time.Duration(float64(sleepSeconds)*0.85) * time.Millisecond)
		}
	}

	go fun()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for {
		select {
		case <-c:
			os.Exit(1)
			return
		case <-error:
			fmt.Println("错误-重新执行")
			time.Sleep(5 * time.Second)
			go fun()
		}
	}
}
