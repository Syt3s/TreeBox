export const QUESTION_STATS_REFRESH_EVENT = "treebox:question-stats-refresh"

export function emitQuestionStatsRefresh() {
  if (typeof window === "undefined") {
    return
  }

  window.dispatchEvent(new Event(QUESTION_STATS_REFRESH_EVENT))
}
