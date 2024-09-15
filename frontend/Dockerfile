# Use the official Node.js image as the base image
FROM node:20-alpine

ARG BACKEND_URL
ARG GITHUB_USERNAME

ENV NEXT_PUBLIC_BACKEND_URL=${BACKEND_URL}
ENV NEXT_PUBLIC_GITHUB_USERNAME=${GITHUB_USERNAME}

# Set the working directory inside the container
WORKDIR /app

# Copy the package.json and install dependencies
COPY package.json ./
RUN npm install

# Copy the rest of the application code
COPY . .

# Build the Next.js application
RUN npm run build

# Expose the port the app runs on
EXPOSE 3000

# Command to run the Next.js application
CMD ["npm", "start"]
