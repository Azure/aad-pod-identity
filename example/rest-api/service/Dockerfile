# Use a lean image
FROM ubuntu:19.04

# Set the working directory to /app
WORKDIR /app

# Copy the script into the container at /app
ADD script.sh /app

# Install curl + pq (to parse JSON)
RUN apt-get install curl
RUN apt-get install pq

# Define environment variable
ENV NAME World

# Run shell script
CMD ["sh", "script.sh"]