-- name: InsertAssestPrice :exec

INSERT INTO assest_price
    (id, assest_id, timestamp, price )
VALUES
    (?, ?, ?, ? );

-- name: GetAllAssestsPrice :many
SELECT * FROM assest_price;


-- name: GetAssestPriceInTimeRange :many
SELECT price FROM assest_price WHERE
    assest_id = @assest_id AND
    timestamp >= @minTimestamp AND
    timestamp <= @maxTimestamp;
