if cfg.NoDuplicates {
    if err := CheckForDuplicates(items); err != nil {
        return fmt.Errorf("duplicate check failed: %w", err)
    }
}
