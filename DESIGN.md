Functional Requirements:

What should /shorten endpoint accept? (POST body format?)
It should accept the long url , custom code (in json format), it should also return(POST) a json response containing the shortened url , shortcode , longURL and expiresAt.

What should /:shortCode endpoint return? (Redirect? JSON?)
It should redirect to the original LongUrl. 
301 - if moved permanently . If code is not found then error 404.

Should we allow custom short codes (e.g., bit.ly/my-custom-url)?
custom short code hmm...Yeah I think it is a good feature as people can relate to what the url is. But there can be multiple short codes that way. So we should try to keep it unique across the database ( by adding numbers at the end / special characters )

What's the maximum URL length we'll accept? (Consider browser limits)
<=2048 - is compatabile with all clients.



Non-Functional Requirements:

How many URLs do we expect to store? (1M? 100M? 1B?)
For now lets start at a lower scale at 1K then scale it to 10K , slowly we can scale it based on the requirement. If we will scale to 10M+ then we should consider sharding/partitioning , consistent hashing or a distributed key-value store.

What's our read:write ratio? (Estimate: 100:1 reads?)
What does this ratio signify ?

What's acceptable latency? (Target: <10ms for redirects)
<10ms of course.

Do we need to handle duplicate URLs? (Same long URL shortened twice)?
If a shortened url is already present for the long url , send it to the user.

What should short_code length be? (Hint: Calculate combinations)
6 is a good length. If we conside base62 encoding then there will be around 56.8 billion varities.

Do we need an index on long_url? Why/why not?
Definetly , when a user provides a long url to shorten it then we should have an index that does a quick lookup and if the long url already exists then it provides the shortened url for it.

Should we add user_id for multi-tenancy? (Not today, but design for it)
User-id ? Well I am not sure why we even need to add user id. Maybe we can add user-id for custom urls ?