local key = KEYS[1]
local now = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]
local should_add = tonumber(ARGV[5])

local cutoff = now - window_ms
redis.call("ZREMRANGEBYSCORE", key, 0, cutoff)

local count = redis.call("ZCARD", key)

local function compute_reset_seconds()
	local oldest = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
	if oldest[2] ~= nil then
		local reset_ms = (tonumber(oldest[2]) + window_ms) - now
		if reset_ms < 0 then
			reset_ms = 0
		end
		return math.ceil(reset_ms / 1000)
	end
	return math.ceil(window_ms / 1000)
end

if should_add == 1 and count < limit then
	redis.call("ZADD", key, now, member)
	redis.call("PEXPIRE", key, window_ms)

	local new_count = count + 1
	local remaining = limit - new_count
	local reset_seconds = compute_reset_seconds()
	return {1, new_count, remaining, reset_seconds}
end

local reset_seconds = compute_reset_seconds()
local remaining = limit - count
if remaining < 0 then
	remaining = 0
end

if count < limit then
	return {1, count, remaining, reset_seconds}
end

return {0, count, 0, reset_seconds}
