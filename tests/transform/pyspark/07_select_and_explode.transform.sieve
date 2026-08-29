// JSON select + explode combined in same section
select_and_explode
    from s3
        bucket orders
        region us-east-1
        prefix enriched
        format delta

    extract
        json select message
            order_id order_id string
            customer_id customer_id string
            total total decimal(18,2)

    extract
        json explode message.line_items array
            product_id product_id string
            item_price item_price decimal(18,2)
            item_qty item_qty bigint