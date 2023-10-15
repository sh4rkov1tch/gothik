# Basing our container on the latest golang container
FROM golang

# Setting gothik as the work directory
WORKDIR gothik/

# Copying the mod and sum files
COPY go.mod go.sum ./ 

# Downlading the modules
RUN go mod download

# Copying source files
COPY *.go ./

# Compiling the app
RUN CGO_ENABLED=0 GOOS=linux go build -o /gothik

# Running it 
CMD ["/gothik"]

