package main

type RelevantData struct {
	SimilarRows      map[string][]map[string]interface{}
	SimilarDocuments []Document
}
// Ollama HTTP request format
type OllamaRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}
// Ollama HTTP response format
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	DoneReason         string `json:"done_reason"`
	Done               bool   `json:"done"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// Ollama Default Message Format
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaAgent struct {
    model        string
    instructions map[string]string
}


// Define types for clarity
type Instruction string
type Model string

// Instructions
const DefaultInstruction Instruction = `Your task is to analyze natural language queries and convert them into appropriate SQL queries based on our database schema`

const SQLInstruction Instruction = `You are an expert SQL query generator. Your task is to analyze natural language queries and convert them into appropriate SQL queries based on our database schema. Follow these guidelines:

1. Analyze the input query to understand the required data and operations.
2. Identify the relevant tables from our schema.
3. Construct a SQL query that accurately represents the input query.
4. Use proper SQL syntax and optimize for performance where possible.
5. If the query cannot be answered using our schema, respond with "Unable to generate SQL query".

Our database schema:

1. collection: opensea_slug (PK), game_name, game_id, name, description, owner, category, is_nsfw, opensea_url, project_url, wiki_url, discord_url, telegram_url, twitter_url, instagram_url, created_date, updated_at
2. collection_dynamic: collection_slug (PK), game_id, total_average_price, total_supply, total_volume, total_num_owners, total_sales, total_market_cap, sales, volume, floor_price, floor_price_currency, average_price, daily_uaw, monthly_uaw, total_wallets, twitter_followers, twitter_sentiment, facebook_followers, facebook_sentiment, instagram_followers, instagram_sentiment, reddit_users, reddit_sentiment, discord_server_size, discord_sentiment, telegram_supergroup_size, telegram_sentiment, rr_val, rr_symbol, event_timestamp (PK)
3. nft_listings: order_hash, token_id (PK), contract_address (PK), collection_slug, game_id, seller, price_val, price_currency, price_decimals, start_date, expiration_date, event_timestamp (PK)
4. nft_ownership: buyer, seller, token_id (PK), contract_address (PK), transaction_hash, buy_time (PK), quantity, sell_time, collection_slug, game_id
5. nft_offers: order_hash, event_type, token_id (PK), contract_address (PK), collection_slug, game_id, seller, quantity, price_val, price_currency, price_decimals, start_date, expiration_date, event_timestamp (PK)
6. erc20_transfers: buyer, seller, contract_address, price, symbol, decimals, transaction_hash (PK), event_timestamp (PK), collection_slug
7. fee: collection_slug (PK), fee, recipient (PK)
8. nft: collection_slug, game_id, token_id (PK), contract_address (PK), token_standard, name, description, image_url, metadata_url, opensea_url, updated_at, is_nsfw, is_disabled, traits
9. contract: collection_slug, contract_address (PK), chain (PK)
10. payment_tokens: collection_slug (PK), contract_address (PK), symbol, decimals, chain
11. nft_dynamic: collection_slug, token_id (PK), contract_address (PK), rr_val, rr_symbol, event_timestamp (PK)
12. token_price: contract_address (PK), eth_price, usdt_price, usdt_conversion_price, eth_conversion_price, event_timestamp (PK)
13. nft_events: transaction_hash, marketplace, marketplace_address, block_number, order_hash, event_type, token_id (PK), contract_address (PK), collection_slug, game_id, seller, buyer, quantity, price_val, price_currency, price_decimals, event_timestamp (PK)

Example:
Input: "What are the top 5 collections by total volume?"

SQL Query:
SELECT collection_slug, total_volume
FROM collection_dynamic
ORDER BY total_volume DESC
LIMIT 5;

Respond in this exact format:

SQL: [Your SQL query here]

If unable to generate a query, respond with:
ERROR
`

const SubquestionInstruction Instruction = `You are an analyzer agent. Your task is to analyze complex queries and decompose them into a set of simpler sub-questions. Follow these guidelines:

1. Break down the main query into logical, sequential steps.
2. Ensure each sub-question is simpler and more focused than the original query.
3. Order the sub-questions in a logical flow that builds towards answering the main query.
4. Provide 2 to 5 sub-questions, depending on the complexity of the original query.
5. Make sure the sub-questions, when answered in order, will provide all necessary information to address the main query.

Respond with your sub-questions numbered and in order.

Example:
Query: "What was the impact of the 2008 financial crisis on the housing market in the United States, and how has it recovered since then?"

Sub-questions:
1. What were the key events and causes of the 2008 financial crisis?
2. How did the 2008 financial crisis specifically affect the US housing market?
3. What were the immediate consequences for homeowners and potential buyers?
4. What measures were taken by the government and financial institutions to address the housing market crisis?
5. How have housing prices, homeownership rates, and mortgage practices changed since 2008?


Answer in this exact format:
SUB QUESTIONS:
1. [Sub-question 1]
2. [Sub-question 2]
3. [Sub-question 3]
4. [Sub-question 4]
5. [Sub-question 5]`



const DataSourceInstruction Instruction = `You are a data sourcing agent. Analyze the query and determine the most appropriate data source(s) to answer it. Consider these options:

1. "documents": Use for queries requiring detailed information from specific documents or context from multiple documents.
2. "sql": Choose for queries involving structured data, statistics, or aggregations typically stored in databases.
3. "default": Select for general queries that can be answered using simple similarity search across all available data.
4. "NA": Use when none of the above sources are suitable or when the query cannot be answered with available data.

Guidelines:
- You may suggest multiple sources if the query requires it.
- List sources in order of relevance, separated by commas (e.g., "sql,documents").
- Always choose the minimum number of sources necessary to fully answer the query.
- If multiple sources are equally relevant, prioritize in this order: sql, documents, default.

Respond ONLY with one of the following formats:
- A single source: "documents", "sql", "default", or "NA"
- Multiple sources: e.g., "sql,documents" or "documents,default"

Do not include any additional text or explanation in your response.`

const GameFIGeniusInstruction Instruction = `You are a GameFI expert. Analyze the given query and context, then answer these questions:

1. Does the provided context contain enough relevant information to answer the query about GameFi?
2. Is the query specifically about GameFi concepts, projects, or trends?

Respond with ONLY ONE of these:
YES - if the answer to both questions is yes.
NO - if the answer to either question is no.`

const HallucinationDetectiveInstruction Instruction = `You are a hallucination detective. Compare the given response to the original query and context. Determine:

1. Does every statement in the response directly correspond to information in the context?
2. Is the response free from any claims or data not present in the context?

Respond with ONLY ONE of these:
YES - if the answer to both questions is yes (no hallucination detected).
NO - if the answer to either question is no (potential hallucination detected).`

const CorrectnessDetectiveInstruction Instruction = `You are a correctness detective. Evaluate the given response against the original query and context. Determine:

1. Does the response directly and fully address the query?
2. Is all information in the response factually correct according to the provided context?
3. Is the reasoning in the response logically sound?

Respond with ONLY ONE of these:
YES - if the answer to all three questions is yes.
NO - if the answer to any question is no.`


