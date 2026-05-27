-- Seed four Premier League teams matching PDF Figure 1.a / 2.a.
-- Strengths are tuned so Chelsea is strongest and Liverpool weakest, mirroring
-- the table ordering shown in the case PDF. Values are dimensionless
-- multipliers centered around 1.0; see internal/domain/team.go.
INSERT INTO teams (name, short_name, attack_strength, defense_strength, home_advantage) VALUES
    ('Chelsea',         'CHE', 1.30, 1.20, 1.30),
    ('Arsenal',         'ARS', 1.05, 1.05, 1.30),
    ('Manchester City', 'MCI', 1.00, 1.00, 1.30),
    ('Liverpool',       'LIV', 0.85, 0.85, 1.30);
