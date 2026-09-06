// Computed column: quantity * price
computed_multiply
    from s3
        bucket sales
        region us-east-1
        prefix line_items
        format delta

    extract
        json explode message.items array
            unit_price unit_price decimal(18,2)
            quantity quantity bigint
            quantity * unit_price total decimal(18,2)
            discount discount decimal(5,2)