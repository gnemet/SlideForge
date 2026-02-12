import os
import zipfile
import shutil
import json
# from lxml import etree

SOURCE_DIR = "/home/gnemet/SlideForgeFiles/source"
UNPACK_DIR = "/home/gnemet/SlideForgeFiles/unpack"
SEED_DIR = "/home/gnemet/SlideForgeFiles/seed"
METADATA_DIR = "/home/gnemet/SlideForgeFiles/metadata"
OUTPUT_DIR = "/home/gnemet/SlideForgeFiles/output"

def setup_dirs():
    for d in [UNPACK_DIR, SEED_DIR, METADATA_DIR, OUTPUT_DIR]:
        if not os.path.exists(d):
            os.makedirs(d)

def unzip_all():
    print(f"Unzipping files from {SOURCE_DIR} to {UNPACK_DIR}...")
    for filename in os.listdir(SOURCE_DIR):
        if filename.endswith(".pptx"):
            file_path = os.path.join(SOURCE_DIR, filename)
            # Create a subfolder for each pptx to avoid collision
            target_subfolder = os.path.join(UNPACK_DIR, filename.replace(".pptx", ""))
            if not os.path.exists(target_subfolder):
                os.makedirs(target_subfolder)
            
            with zipfile.ZipFile(file_path, 'r') as zip_ref:
                zip_ref.extractall(target_subfolder)
            print(f"  Unpacked: {filename}")

def analyze_files():
    print("Analyzing unpacked files...")
    # This is a placeholder for actual analysis logic
    # We could look for specific placeholders or slide counts
    for folder in os.listdir(UNPACK_DIR):
        folder_path = os.path.join(UNPACK_DIR, folder)
        if os.path.isdir(folder_path):
            slides_path = os.path.join(folder_path, "ppt", "slides")
            if os.path.exists(slides_path):
                slide_count = len([f for f in os.listdir(slides_path) if f.startswith("slide")])
                print(f"  {folder}: {slide_count} slides")

def create_seed():
    print("Creating seed PPTX...")
    # Pick one pptx to be the base for the seed
    pptx_files = [f for f in os.listdir(SOURCE_DIR) if f.endswith(".pptx")]
    if not pptx_files:
        print("No PPTX files found to create seed.")
        return
    
    first_pptx = pptx_files[0]
    seed_path = os.path.join(SEED_DIR, "seed.pptx")
    shutil.copy2(os.path.join(SOURCE_DIR, first_pptx), seed_path)
    print(f"  Created seed.pptx from {first_pptx}")

def create_metadata():
    print("Creating test metadata...")
    metadata = {
        "client_name": "Test Client Zrt.",
        "project_date": "2026-02-11",
        "offer_value": "10,000,000 HUF",
        "author": "Antigravity AI"
    }
    metadata_path = os.path.join(METADATA_DIR, "test_metadata.json")
    with open(metadata_path, 'w', encoding='utf-8') as f:
        json.dump(metadata, f, indent=4, ensure_ascii=False)
    print(f"  Created {metadata_path}")

def generate_new_pptx():
    print("Generating new PPTX from seed and metadata...")
    seed_path = os.path.join(SEED_DIR, "seed.pptx")
    metadata_path = os.path.join(METADATA_DIR, "test_metadata.json")
    output_path = os.path.join(OUTPUT_DIR, "generated_offer.pptx")
    
    if not os.path.exists(seed_path) or not os.path.exists(metadata_path):
        print("Seed or metadata missing.")
        return

    with open(metadata_path, 'r', encoding='utf-8') as f:
        metadata = json.load(f)

    # Simple implementation: unzip seed, replace text in XML, re-zip
    temp_work_dir = os.path.join(OUTPUT_DIR, "temp_work")
    if os.path.exists(temp_work_dir):
        shutil.rmtree(temp_work_dir)
    os.makedirs(temp_work_dir)

    with zipfile.ZipFile(seed_path, 'r') as zip_ref:
        zip_ref.extractall(temp_work_dir)

    # Process XML files
    for root, dirs, files in os.walk(temp_work_dir):
        for file in files:
            if file.endswith(".xml"):
                full_path = os.path.join(root, file)
                with open(full_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                original_content = content
                for key, value in metadata.items():
                    placeholder = "{{" + key + "}}"
                    content = content.replace(placeholder, str(value))
                
                if content != original_content:
                    with open(full_path, 'w', encoding='utf-8') as f:
                        f.write(content)

    # Re-zip
    with zipfile.ZipFile(output_path, 'w', zipfile.ZIP_DEFLATED) as zip_out:
        for root, dirs, files in os.walk(temp_work_dir):
            for file in files:
                abs_path = os.path.join(root, file)
                rel_path = os.path.relpath(abs_path, temp_work_dir)
                zip_out.write(abs_path, rel_path)

    shutil.rmtree(temp_work_dir)
    print(f"  Generated {output_path}")

if __name__ == "__main__":
    setup_dirs()
    unzip_all()
    analyze_files()
    create_seed()
    create_metadata()
    generate_new_pptx()
