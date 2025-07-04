package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http" // JSON取得のために残す
	"net/url"
	"os"   // 標準出力に書き出すために必要
	"time"
)

// JSONデータの構造を定義します
type JSONData struct {
	Pages []struct {
		Name        string    `json:"name"`
		ModTime     time.Time `json:"modTime"`
		Description string    `json:"description"`
		Cover       string    `json:"cover"`
	} `json:"pages"`
}

// RSS 2.0の構造を定義します
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel *Channel `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	LastBuildDate string `xml:"lastBuildDate,omitempty"`
	PubDate       string `xml:"pubDate,omitempty"`
	TTL           int    `xml:"ttl,omitempty"`
	Items         []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate,omitempty"`
	GUID        string `xml:"guid,omitempty"`
}

// JSONを取得するURL
const baseURL = "https://inline.inajob.freeddns.org"
const jsonURL = "https://inline.inajob.freeddns.org/page/twitter-5643382?detail=1"
const itemURLPrefix = "https://inline.inajob.freeddns.org/web?user=twitter-5643382&id=" // RSSアイテムのリンクのベースURL

func main() {
	// 1. URLからJSONデータを取得する
	resp, err := http.Get(jsonURL)
	if err != nil {
		log.Fatalf("JSONの取得エラー: %v", err) // サーバーではないのでFatalで終了
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("JSON取得時にHTTPステータスコードエラー: %d %s", resp.StatusCode, resp.Status)
	}

	jsonDataBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("レスポンスボディの読み込みエラー: %v", err)
	}

	var data JSONData
	if err := json.Unmarshal(jsonDataBytes, &data); err != nil {
		log.Fatalf("JSONのパースエラー: %v", err)
	}

	// 2. RSS構造体を構築する
	now := time.Now().Format(time.RFC1123Z)
	rss := &RSS{
		Version: "2.0",
		Channel: &Channel{
			Title:         "inajob Inline Feed",
			Link:          baseURL,
			Description:   "An RSS feed generated from inline.inajob.freeddns.org pages.",
			LastBuildDate: now,
			PubDate:       now,
			TTL:           60,
			Items:         []Item{},
		},
	}

	// 最初の10件のページのみを処理
	pagesToProcess := data.Pages
	if len(pagesToProcess) > 20 {
		pagesToProcess = pagesToProcess[:20]
	}

	for _, page := range pagesToProcess {
		itemLink := itemURLPrefix + url.PathEscape(page.Name)
		itemGUID := itemLink

		// descriptionに画像を含める処理
		htmlDescription := ""
		if page.Cover != "" {
			parsedCoverURL, err := url.Parse(page.Cover)
                        var resolvedCoverURL string
                        if err != nil {
                            // cover URL自体が無効な場合
			    //log.Printf("警告: cover URL '%s' のパースエラー: %v. 元のURLをそのまま使用します。", page.Cover, err)
                            resolvedCoverURL = page.Cover // パースエラーでも一応元の文字列を使う
                        } else if !parsedCoverURL.IsAbs() {
                            // 相対URLの場合 (IsAbs()がfalseを返す)
                            resolvedCoverURL = baseURL + html.EscapeString(page.Cover)
                        } else {
                            // 絶対URLの場合
                            resolvedCoverURL = page.Cover
                        }

			htmlDescription += fmt.Sprintf("<p><img src=\"%s\" alt=\"%s\" style=\"max-width:100%%; height:auto;\" /></p>\n", resolvedCoverURL, resolvedCoverURL)
		}
		htmlDescription += fmt.Sprintf("<p>%s</p>", html.EscapeString(page.Description))

		finalDescription := fmt.Sprintf("<![CDATA[%s]]>", htmlDescription)

		rss.Channel.Items = append(rss.Channel.Items, Item{
			Title:       page.Name,
			Link:        itemLink,
			Description: finalDescription,
			PubDate:     page.ModTime.Format(time.RFC1123Z),
			GUID:        itemGUID,
		})
	}

	// 3. RSS XMLを生成する
	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		log.Fatalf("RSS XMLの生成エラー: %v", err)
	}

	// 4. 標準出力に書き出す
	// XMLヘッダーを追加
	if _, err := os.Stdout.Write([]byte(xml.Header)); err != nil {
		log.Fatalf("XMLヘッダーの書き込みエラー: %v", err)
	}
	if _, err := os.Stdout.Write(output); err != nil {
		log.Fatalf("RSS XMLの書き込みエラー: %v", err)
	}
}
