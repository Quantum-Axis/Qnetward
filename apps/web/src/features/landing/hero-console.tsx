import Image from "next/image";
import dashboard from "../../../public/landing/dashboard.png";

/** The hero shows the real console: a screenshot of the Overview page, set straight on the page. */
export function HeroConsole() {
  return (
    <div className="relative mx-auto max-w-[1200px]">
      <div className="relative overflow-hidden rounded-[14px] bg-[#111113] shadow-[0_30px_100px_rgba(0,0,0,0.6)] sm:rounded-[24px]">

        <Image
          src={dashboard}
          alt="The Qnetward Overview: target count, average latency, packet loss, and alert rules, a response time chart per target, target health, and the targets table."
          sizes="(max-width: 1240px) 100vw, 1200px"
          placeholder="blur"
          priority
          className="block h-auto w-full"
        />
      </div>
    </div>
  );
}
