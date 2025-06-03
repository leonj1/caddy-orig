# Use an official Python runtime as a parent image
FROM python:3.9-slim

WORKDIR /app

# Copy the requirements file and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application code
COPY app.py .

# Make port 8080 available to the world outside this container
# This is the port uvicorn will listen on by default as configured in CMD.
# Can be overridden by PYECHOSERVER_PORT env var if app.py's __main__ block was used,
# but for direct uvicorn CMD, we set it here.
EXPOSE 8080

# Command to run the application using uvicorn
# The PYECHOSERVER_PORT environment variable can be used by Caddy to tell the container
# which port to listen on if it differs from the default 8080.
# Uvicorn's --port argument will take precedence if PYECHOSERVER_PORT is also set by Caddy.
# For simplicity, we'll rely on the serverless plugin setting the correct port
# and the application inside the container listening on that port.
# The serverless plugin defaults to 8080 if not specified in Caddy config.
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8080", "--log-level", "debug"]
