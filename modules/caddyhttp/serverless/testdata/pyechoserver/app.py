from fastapi import FastAPI, Request, HTTPException
from fastapi.responses import JSONResponse
import uvicorn
import os
import sys
from contextlib import asynccontextmanager
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Track if we've handled a request
request_handled = False

@asynccontextmanager
async def lifespan(app: FastAPI):
    yield
    # This will run on shutdown
    if request_handled:
        print("Request processed, shutting down server")
        sys.exit(0)

app = FastAPI(lifespan=lifespan)

@app.post("/")
async def echo(request: Request):
    global request_handled
    timestamp = datetime.now().isoformat()
    logger.info(f"[{timestamp}] Received POST request to /")

    try:
        logger.info(f"[{datetime.now().isoformat()}] Extracting request headers")
        headers = dict(request.headers)
        logger.info(f"[{datetime.now().isoformat()}] Headers extracted: {len(headers)} headers found")

        logger.info(f"[{datetime.now().isoformat()}] Reading request body")
        body = await request.body()
        logger.info(f"[{datetime.now().isoformat()}] Body read: {len(body)} bytes")

        logger.info(f"[{datetime.now().isoformat()}] Preparing response data")
        response_data = {
            "headers": headers,
            "body": body.decode('utf-8') # Assuming UTF-8, adjust if other encodings are expected
        }

        logger.info(f"[{datetime.now().isoformat()}] Setting request_handled flag to True")
        request_handled = True

        # Schedule shutdown after response is sent
        logger.info(f"[{datetime.now().isoformat()}] Scheduling server shutdown")
        import asyncio
        asyncio.create_task(shutdown_after_delay())

        logger.info(f"[{datetime.now().isoformat()}] Returning JSON response with status 200")
        return JSONResponse(content=response_data, status_code=200)
    except Exception as e:
        error_timestamp = datetime.now().isoformat()
        logger.error(f"[{error_timestamp}] Exception occurred: {str(e)}")
        logger.error(f"[{error_timestamp}] Raising HTTPException with status 500")
        raise HTTPException(status_code=500, detail=str(e))

async def shutdown_after_delay():
    # Small delay to ensure response is sent before shutdown
    await asyncio.sleep(0.5)
    os._exit(0)  # Force exit

# This block is for local development/testing if you run `python app.py`
# In production with Docker, uvicorn will be called directly.
if __name__ == "__main__":
    port = int(os.environ.get("PYECHOSERVER_PORT", 8080))
    uvicorn.run(app, host="0.0.0.0", port=port)