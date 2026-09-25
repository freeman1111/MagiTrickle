<script lang="ts">
  // Fork-only: tells users of this build where to report problems, so they do not go to the original authors
  import { locale } from "../data/locale.svelte";

  import { CircleInfo } from "./ui/icons";

  const FORK_ISSUES = "https://github.com/freeman1111/MagiTrickle/issues";

  const text = $derived(
    locale.state.value === "ru"
      ? "Неофициальный форк MagiTrickle. Об ошибках этой сборки сообщайте в репозиторий форка, а не авторам оригинала:"
      : "Unofficial fork of MagiTrickle. Report problems with this build to the fork repository, not to the original authors:",
  );
</script>

<div class="fork-notice">
  <div class="icon">
    <CircleInfo size={22} />
  </div>
  <div class="text">
    {text}
    <a target="_blank" rel="noopener noreferrer" href={FORK_ISSUES}>{FORK_ISSUES}</a>
  </div>
</div>

<style>
  .fork-notice {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 0.6rem;
    padding-bottom: 0.8rem;
    border-bottom: 1px solid var(--bg-light-extra);
  }

  .icon {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-2);
  }

  .text {
    font-size: smaller;
    min-width: 0;
  }

  a {
    display: block;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--accent);
  }

  a:hover {
    color: var(--text);
  }
</style>
