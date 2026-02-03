---
description: PPTX processing
---

Workflow of PPTX processing

# Folders
## Stage
- upload PPTX here, after processed move away

## Template
- processed PPTX move here

## Thumbnails
- slide pictures one PPTX one folder 

# Processing
## Steps
1. new file uploaded
2. unzip to xml 
3. find human text and collect it by slide, save to database
4. convert xml to json by slide , save to database
5. create images by slide, copy to thumbnails folder 
6. move pptx file to template folder  

## Observer
1. listening Stage folder for new files
