package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"strings"
)

func main() {
	pptxPath := "pptx/MINTA_B_ajánlat_DD_20250916_v5.pptx"
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			fmt.Printf("--- %s ---\n", f.Name)
			rc, err := f.Open()
			if err != nil {
				continue
			}
			dumpSlideXML(rc)
			rc.Close()
		}
	}
}

func dumpSlideXML(r io.Reader) {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch el := tok.(type) {
		case xml.StartElement:
			// Print interesting tags and their attributes
			if el.Name.Local == "cNvPr" || el.Name.Local == "ph" || el.Name.Local == "spPr" || el.Name.Local == "t" || el.Name.Local == "cmAk" {
				fmt.Printf("<%s", el.Name.Local)
				for _, a := range el.Attr {
					fmt.Printf(" %s=%q", a.Name.Local, a.Value)
				}
				fmt.Printf(">\n")
			}
		}
	}
}
