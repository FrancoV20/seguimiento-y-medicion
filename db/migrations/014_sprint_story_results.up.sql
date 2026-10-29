CREATE TABLE sprint_story_results (
    sprint_id    INTEGER     NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    story_id     INTEGER     NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    final_status VARCHAR(20) NOT NULL,
    story_points INTEGER,
    PRIMARY KEY (sprint_id, story_id)
);
