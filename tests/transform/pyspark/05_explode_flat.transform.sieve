// JSON explode with flat fields
explode_flat
    from s3
        bucket inventory
        region us-east-1
        prefix items
        format delta

    extract
        json explode message.items array
            sku sku string
            qty quantity bigint
            price price decimal(18,2)