# Orchestrator

Microservice for orchestrating AI and handling various data sources, processing queries, and managing embeddings for efficient information retrieval.


### Multi-source Data Handling: Integrates data from SQL databases, document storage, and web sources.
### Embeddings Generation: Creates and manages embeddings for both structured (database rows) and unstructured (documents) data.
### Intelligent Query Processing: Utilizes a multi-step approach to break down, answer, and synthesize complex queries.
### File Type Support: Handles various file types including PDF, DOCX, TXT, and Markdown.
### IPFS Integration: Retrieves and processes files stored on IPFS.
### Ollama Integration: Leverages Ollama for natural language processing tasks.

## Key Components

### Database Management (db.go, db_types.go):

Handles connections to PostgreSQL database.
Defines database schemas and operations.


### bold[Embeddings] Processing (embeddings.go):

Generates embeddings for database rows and documents.
Integrates with Ollama for embedding creation.


### File Processing (text_utils.go):

Extracts text from various file types (PDF, DOCX, TXT, Markdown).
Processes and normalizes text data.


### IPFS Integration (ipfs.go):

Retrieves files from IPFS network.


### Query Processing (ollama.go, ollama_rag.go):

Implements a sophisticated query processing pipeline.
Breaks down complex queries into sub-questions.
Utilizes different data sources based on query requirements.


### API Router (router.go):

Sets up HTTP endpoints for the application.
Handles authentication and request routing.



## Setup and Usage

Ensure you have Go installed on your system.
Clone this repository.
Set up environment variables for database connection:

TIMESCALE_ADDRESS
TIMESCALE_USER
TIMESCALE_PASSWORD
TIMESCALE_DATABASE


Install dependencies:
Copygo mod tidy

Run the application:
Copygo run .


### API Endpoints

/ping: Health check endpoint.
/user/:name: Get user profile.
/admin: Admin operations (authenticated).
/generateRowEmbeddings: Generate embeddings for database rows.
/generateDocumentEmbeddings: Generate embeddings for documents.
/llmQuery: Process complex queries using the LLM pipeline.

### Dependencies

github.com/gin-gonic/gin: Web framework
github.com/go-pg/pg/v10: PostgreSQL ORM
github.com/pgvector/pgvector-go: Vector operations for PostgreSQL
github.com/tmc/langchaingo/llms/ollama: Ollama integration
Various other libraries for file processing and utilities

### Contributing
Contributions to this project are welcome. Please ensure you follow the existing code style and add unit tests for any new functionality.
MIT LICENSE
