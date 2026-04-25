package ch.rasc.timefold.demo;

import java.util.Arrays;
import java.util.concurrent.CountDownLatch;

import ch.rasc.timefold.demo.domain.FilmFestivalSchedule;
import ch.rasc.timefold.demo.web.FestivalScheduleServer;
import ch.rasc.timefold.demo.web.ScheduleResponse;

public final class FilmFestivalScheduleApp {

  private FilmFestivalScheduleApp() {
  }

  public static void main(String[] args) throws Exception {
    FilmFestivalSchedule solution = FilmFestivalScheduleSupport.solveSampleFestival();
    FilmFestivalScheduleSupport.printSolution(solution);

    if (Arrays.asList(args).contains("--console")) {
      return;
    }

    int port = Integer.getInteger("festival.port", 8080);
    FestivalScheduleServer server = new FestivalScheduleServer(port, ScheduleResponse.fromSolution(solution));
    Runtime.getRuntime().addShutdownHook(new Thread(server::stop));
    server.start();

    System.out.println();
    System.out.println("Festival web UI: http://localhost:" + port + "/");
    System.out.println("Festival JSON API: http://localhost:" + port + "/api/schedule");
    new CountDownLatch(1).await();
  }
}