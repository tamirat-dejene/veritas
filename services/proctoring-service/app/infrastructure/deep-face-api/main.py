from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
# pyrefly: ignore [missing-import]
from deepface import DeepFace
import base64, tempfile, os, numpy as np

@asynccontextmanager
async def lifespan(app):
    DeepFace.build_model("Facenet512")
    yield

app = FastAPI(lifespan=lifespan)

def b64_to_file(b64_str: str) -> str:
    data = base64.b64decode(b64_str)
    tmp = tempfile.NamedTemporaryFile(delete=False, suffix=".jpg")
    tmp.write(data)
    tmp.close()
    return tmp.name

def cosine_distance(a, b):
    a, b = np.array(a), np.array(b)
    return float(1 - np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b)))

class RepresentRequest(BaseModel):
    img: str
    model_name: str = "Facenet512"
    detector_backend: str = "opencv"

class VerifyRequest(BaseModel):
    img1: str
    img2: str
    model_name: str = "Facenet512"
    detector_backend: str = "opencv"

class EmbedRequest(BaseModel):
    img: str
    model_name: str = "Facenet512"
    detector_backend: str = "retinaface"

class CompareRequest(BaseModel):
    embedding: list
    img: str
    model_name: str = "Facenet512"
    detector_backend: str = "opencv"
    threshold: float = 0.30

class AnalyzeRequest(BaseModel):
    img_base64: str
    actions: list = ["age", "gender", "emotion", "race"]

@app.post("/represent")
def represent(req: RepresentRequest):
    f = b64_to_file(req.img)
    try:
        results = DeepFace.represent(
            img_path=f,
            model_name=req.model_name,
            detector_backend=req.detector_backend,
            enforce_detection=True
        )
        return {"results": results}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
    finally:
        os.unlink(f)

@app.post("/embed")
def embed(req: EmbedRequest):
    """Call once at registration. Uses retinaface for accuracy. Store the returned embedding."""
    f = b64_to_file(req.img)
    try:
        results = DeepFace.represent(
            img_path=f,
            model_name=req.model_name,
            detector_backend=req.detector_backend,
            enforce_detection=True
        )
        return {"embedding": results[0]["embedding"]}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
    finally:
        os.unlink(f)

@app.post("/compare")
def compare(req: CompareRequest):
    """Fast path: pass stored embedding + live frame. No re-embedding of reference."""
    f = b64_to_file(req.img)
    try:
        results = DeepFace.represent(
            img_path=f,
            model_name=req.model_name,
            detector_backend=req.detector_backend,
            enforce_detection=True
        )
        probe_embedding = results[0]["embedding"]
        distance = cosine_distance(req.embedding, probe_embedding)
        return {
            "verified": distance < req.threshold,
            "distance": round(distance, 4),
            "threshold": req.threshold,
            "model": req.model_name
        }
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
    finally:
        os.unlink(f)

@app.post("/verify")
def verify(req: VerifyRequest):
    f1, f2 = b64_to_file(req.img1), b64_to_file(req.img2)
    try:
        result = DeepFace.verify(
            img1_path=f1,
            img2_path=f2,
            model_name=req.model_name,
            detector_backend=req.detector_backend,
            enforce_detection=True
        )
        return {
            "verified":  result["verified"],
            "distance":  result["distance"],
            "threshold": result["threshold"],
            "model":     result["model"],
        }
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
    finally:
        os.unlink(f1)
        os.unlink(f2)

@app.post("/analyze")
def analyze(req: AnalyzeRequest):
    f = b64_to_file(req.img_base64)
    try:
        result = DeepFace.analyze(f, actions=req.actions, enforce_detection=False)
        return result
    finally:
        os.unlink(f)

@app.get("/health")
def health():
    return {"status": "ok"}