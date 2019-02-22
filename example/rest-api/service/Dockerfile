# Use a lean image
FROM appropriate/curl

# Set the working directory to /app
WORKDIR /app

# Copy the script into the container at /app
ADD script.sh /app

# Install pq (to parse JSON)
RUN apk add --update pq && rm -rf /var/cache/apk/*

# Define environment variable
ENV NAME World

# Run shell script
CMD ["bash", "script.sh"]