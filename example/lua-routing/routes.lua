-- Changelog:
-- v0.1: simple routing decision based on map and basic caching

-- Global cache for parsed route data
-- TODO: might need a "purge" strategy to avoid memory leak
local cache = {}

-- Parses a route string like "v1:70,v2:20,v3:10" into a fast, usable format.
-- This is the "heavy" work that only runs when a route's config changes.
local function parse_route(route_str)
    local routes = {}
    local total_weight = 0
    local cumulative_weight = 0

    -- Split the string by comma to get each backend part
    for part in string.gmatch(route_str, "([^,]+)") do
        -- Split by colon to get backend name and its weight
        local backend, weight_str = string.match(part, "(.+):(%d+)")
        if backend and weight_str then
            local weight = tonumber(weight_str)
            total_weight = total_weight + weight
            cumulative_weight = cumulative_weight + weight
            -- Store the backend with its CUMULATIVE weight. This is key for fast lookups.
            table.insert(routes, { backend = backend, cumulative = cumulative_weight })
        end
    end

    return { routes = routes, total = total_weight }
end

-- Main function called by HAProxy for each request
function routesv4(txn)
    -- The first argument passed from HAProxy is the route configuration string
    local route_str = txn.f:var("txn.routes")
    if not route_str or route_str == "" then return end

    -- ⚡ THE CACHING LOGIC
    -- If the route string isn't in our cache, parse it and store it.
    if not cache[route_str] then
        cache[route_str] = parse_route(route_str)
    end

    local route_data = cache[route_str]
    if not route_data or route_data.total == 0 then return end

    -- Generate a random number based on the total weight for this route
    -- Note: rand(n) generates 0 to n-1, so we compare with '<'
    local random_val = txn.f:rand(route_data.total)

    -- Find the correct backend by checking the random value against the cumulative weights
    for _, route in ipairs(route_data.routes) do
        if random_val < route.cumulative then
            txn.set_var(txn, 'txn.backend', route.backend)
            txn.set_var(txn, 'txn.random', random_val)
            return
        end
    end
end

-- Register the function to be called from HAProxy
core.register_action("routesv4", { "http-req" }, routesv4)
