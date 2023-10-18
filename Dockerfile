# Create image based on the official Node image from dockerhub
FROM ghcr.io/puppeteer/puppeteer:21.3.8

# Use a non-root user to run all the commands
USER node

WORKDIR /home/node

# Copy dependency definitions
COPY --chown=node:node package.json ./package.json
COPY --chown=node:node package-lock.json ./package-lock.json
 
# Install dependencies
RUN npm ci

# Get all the code needed to run the app. TODO: Figure out a way to only copy what's needed.
COPY --chown=node:node . .

# Necessary to make sure that the dbus is running
ENV DBUS_SESSION_BUS_ADDRESS autolaunch:

CMD ["npm", "start"]
# ENTRYPOINT ["tail", "-f", "/dev/null"]
