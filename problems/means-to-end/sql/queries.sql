-- name: InsertAssestPrice :exec

INSERT INTO assest_price
    (id, assest_id, timestamp, price )
VALUES
    (?, ?, ?, ? );

-- name: GetAllAssestsPrice :many
SELECT * FROM assest_price;
