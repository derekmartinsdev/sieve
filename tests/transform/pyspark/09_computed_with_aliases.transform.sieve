// Computed column referencing already-aliased fields
computed_with_aliases
    from s3
        bucket warehouse
        region us-west-2
        prefix shipments
        format delta

    extract
        json explode message.packages array
            weight_kg weight decimal(18,4)
            rate_per_kg rate decimal(18,4)
            weight * rate shipping_cost decimal(18,2)