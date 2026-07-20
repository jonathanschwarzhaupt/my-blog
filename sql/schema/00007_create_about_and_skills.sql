-- +goose Up
CREATE TABLE about_revision (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX about_revision_created_at_idx ON about_revision (created_at DESC, id DESC);

CREATE TABLE skill (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category text NOT NULL,
    name text NOT NULL,
    order_key double precision NOT NULL
);

CREATE INDEX skill_order_key_idx ON skill (order_key ASC, id ASC);

-- Seed data: the About page's content used to be hardcoded in
-- ui/templ/pages/blog/about.templ. This carries it over as the first
-- revision / first skill rows, so the page isn't blank the moment this
-- migration runs — see docs/adr/0009-about-page-db-backed-with-revision-history.md.
INSERT INTO about_revision (body) VALUES ($$Hi there! 👋 I'm Jonathan, a DevOps Engineer in healthtech, based in Munich. I build reliable infrastructure and software for a living — and then, for reasons I haven't fully examined, come home and build more of it for fun.

## My Journey

I started out in business, completing both my bachelor's and master's degrees in business studies. During my MSc in Business Analytics, I found out I liked the technical side a lot more than I expected, and spent the following years as a Data Engineer, architecting cloud data platforms on Microsoft Azure and AWS.

Over the last year and a half, I deliberately deepened my infrastructure skills — Kubernetes, GitOps, the foundational layer everything else depends on — because I wanted to work closer to where reliability actually gets built. That effort became a DevOps Engineer role at Avelios Medical, where I work on the Kubernetes and GitOps infrastructure behind a hospital information system (KIS) — real, mission-critical software, on a team I very much had to earn my way onto. The business background hasn't gone away; it's still how I think. I'm just building the layer underneath everything else now, instead of the layer on top of it.

## What I Do

At Avelios Medical, I work on the foundational infrastructure that lets teams ship reliably and our software function the way hospital software has to — Kubernetes, GitOps (ArgoCD, specifically), CI/CD, observability. "Mostly working" isn't a real option in healthtech, which is exactly the kind of pressure that makes the work interesting. My data engineering background still shapes how I approach it: pipelines and platforms live and die by the same things — clear boundaries, good automation, and infrastructure that evolves without someone babysitting it.

I bring the same cloud-native instinct from my Azure/AWS data platform days to infrastructure now — picking the right tool for the problem, managed or self-hosted, rather than defaulting to one camp out of habit.

## Beyond Engineering

Off the clock, you'll find me on the field hockey pitch — I've played competitively for 15+ years — or out exploring Munich's golf courses and its neighborhoods.

The infrastructure habit doesn't really turn off, though. I self-host an Apache Arrow Flight SQL data warehouse in my homelab, with pipelines pulling in my own bank transactions — because apparently a budgeting app was never going to be enough for me. Curious how the Flight SQL server is set up? I [wrote it up](/posts/gizmosql-in-kubernetes). And yes, this blog itself runs on Kubernetes — GitOps habits die hard, even for a personal site with a readership of, generously, a handful.

It's also my sandbox for something else I've been deliberately leaning into: agentic AI coding — letting AI operate with real autonomy at the layers I'm comfortable handing over, under constraints and supervision I set myself. I'm planning a post series on what that's actually looked like in practice, mistakes included — for now, consider this whole site the first data point.

If you're in the middle of a similar business-to-tech jump, or just want to talk shop about Kubernetes, GitOps, or data platforms, reach out — I remember exactly what that jump looks like from the other side, and I'm always happy to help. Otherwise, have a look at what I've [written](/posts) or [built](/projects).$$);

INSERT INTO skill (category, name, order_key) VALUES
    ('Infrastructure & Platform', 'Kubernetes', 1),
    ('Infrastructure & Platform', 'GitOps (ArgoCD, Flux)', 2),
    ('Infrastructure & Platform', 'Docker', 3),
    ('Infrastructure & Platform', 'CI/CD', 4),
    ('Cloud', 'Microsoft Azure (Certified Data Engineer & Administrator)', 5),
    ('Cloud', 'AWS', 6),
    ('Data Tools', 'Airflow', 7),
    ('Data Tools', 'DuckDB', 8),
    ('Data Tools', 'dlt', 9),
    ('Data Tools', 'dbt', 10),
    ('Languages', 'Python', 11),
    ('Languages', 'SQL', 12),
    ('Languages', 'Go', 13),
    ('Languages', 'Bash', 14);

-- +goose Down
DROP TABLE IF EXISTS skill;
DROP TABLE IF EXISTS about_revision;
