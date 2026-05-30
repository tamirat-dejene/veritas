import base64
import os
import urllib.request
import requests

API_URL = "https://deepface-api-xxxxxxxxxxxx.us-central1.run.app"

def download_image(url: str, filename: str):
    if not os.path.exists(filename):
        headers = {'User-Agent': 'Mozilla/5.0'}
        response = requests.get(url, headers=headers)
        response.raise_for_status()
        with open(filename, 'wb') as f:
            f.write(response.content)

def get_base64_string(filepath: str) -> str:
    with open(filepath, "rb") as f:
        return base64.b64encode(f.read()).decode('utf-8')

def run_evaluation():
    print(f"Connecting to DeepFace API at {API_URL}...")
    
    test_imgs_dir = "/home/tamirat-dejene/Documents/coursework/final_project/veritas/services/proctoring-service/app/infrastructure/deep-face-api/test_imgs"
    
    images = []
    for i in range(1, 17):
        path = os.path.join(test_imgs_dir, f"{i}.png")
        if os.path.exists(path):
            images.append({"id": f"{i}.png", "path": path})

    print(f"\n[1] Generating embeddings for {len(images)} images...")
    for img in images:
        b64 = get_base64_string(img["path"])
        img["b64"] = b64
        res = requests.post(f"{API_URL}/embed", json={
            "img": b64, 
            "model_name": "Facenet512", 
            "detector_backend": "retinaface"
        })
        if res.status_code == 200:
            img["embedding"] = res.json()["embedding"]
        else:
            print(f"Failed to embed {img['id']}: {res.text}")
            img["embedding"] = None

    print(f"\n[2] Performing NxN comparisons (Unique Pairs)...")
    matches = []
    for i in range(len(images)):
        for j in range(i + 1, len(images)):
            img_a = images[i]
            img_b = images[j]
            
            if not img_a.get("embedding") or not img_b.get("b64"):
                continue
                
            res_compare = requests.post(f"{API_URL}/compare", json={
                "embedding": img_a["embedding"],
                "img": img_b["b64"],
                "model_name": "Facenet512",
                "detector_backend": "opencv",
                "threshold": 0.30
            })
            
            if res_compare.status_code == 200:
                data = res_compare.json()
                if data["verified"]:
                    matches.append((img_a["id"], img_b["id"], data["distance"]))
            else:
                print(f"Failed to compare {img_a['id']} vs {img_b['id']}")

    print("\n[3] Results (Matches Found):")
    if matches:
        for a, b, dist in matches:
            print(f"  ✅ {a} & {b} are the SAME person! (Distance: {dist:.4f})")
    else:
        print("  ❌ No matches found among any of the images. They are all different people.")

    print("\n🎉 Pairwise Evaluation Completed!")

if __name__ == "__main__":
    run_evaluation()
