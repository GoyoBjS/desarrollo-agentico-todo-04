// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var completed = task.classList.toggle("task-completed");
  var button = form.querySelector("button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }

  fetch(form.action);
});

// Reordena las tareas arrastrando el asa (ratón, táctil o lápiz) o con las flechas del teclado.
(function () {
  var queue = Promise.resolve();
  var drag = null;

  function indexOf(li) {
    return Array.prototype.indexOf.call(li.parentNode.children, li) + 1;
  }

  function announce(li) {
    var status = document.querySelector("[data-reorder-status]");
    if (!status) {
      return;
    }
    var title = li.querySelector(".task-title").textContent;
    status.textContent = title + ": posición " + indexOf(li) + " de " + li.parentNode.children.length;
  }

  function persist(li) {
    var form = li.querySelector("[data-task-move]");
    form.elements.position.value = String(indexOf(li));
    var body = new FormData(form);

    queue = queue.then(function () {
      return fetch(form.action, { method: "POST", body: body, credentials: "same-origin" }).then(function (response) {
        if (!response.ok) {
          window.location.reload();
        }
      });
    }).catch(function () {
      window.location.reload();
    });

    announce(li);
  }

  document.addEventListener("pointerdown", function (event) {
    var handle = event.target.closest("[data-task-handle]");
    if (!handle || event.button !== 0) {
      return;
    }

    event.preventDefault();
    handle.setPointerCapture(event.pointerId);

    var li = handle.closest(".task");
    drag = {
      li: li,
      handle: handle,
      pointerId: event.pointerId,
      startIndex: indexOf(li),
      nextSibling: li.nextElementSibling
    };
    li.classList.add("task-dragging");
  });

  document.addEventListener("pointermove", function (event) {
    if (!drag || event.pointerId !== drag.pointerId) {
      return;
    }

    var li = drag.li;
    var list = li.parentNode;
    var siblings = list.children;
    var target = null;

    for (var i = 0; i < siblings.length; i++) {
      if (siblings[i] === li) {
        continue;
      }
      var rect = siblings[i].getBoundingClientRect();
      if (event.clientY < rect.top + rect.height / 2) {
        target = siblings[i];
        break;
      }
    }

    if (target) {
      if (li.nextElementSibling !== target) {
        list.insertBefore(li, target);
      }
    } else if (list.lastElementChild !== li) {
      list.appendChild(li);
    }
  });

  document.addEventListener("pointerup", function (event) {
    if (!drag || event.pointerId !== drag.pointerId) {
      return;
    }

    var li = drag.li;
    var moved = indexOf(li) !== drag.startIndex;
    li.classList.remove("task-dragging");
    drag = null;

    if (moved) {
      persist(li);
    }
  });

  document.addEventListener("pointercancel", function (event) {
    if (!drag || event.pointerId !== drag.pointerId) {
      return;
    }

    var li = drag.li;
    li.classList.remove("task-dragging");
    li.parentNode.insertBefore(li, drag.nextSibling);
    drag = null;
  });

  document.addEventListener("keydown", function (event) {
    var handle = event.target.closest && event.target.closest("[data-task-handle]");
    if (!handle) {
      return;
    }

    var li = handle.closest(".task");
    var list = li.parentNode;

    if (event.key === "ArrowUp") {
      var prev = li.previousElementSibling;
      if (!prev) {
        return;
      }
      event.preventDefault();
      list.insertBefore(li, prev);
    } else if (event.key === "ArrowDown") {
      var next = li.nextElementSibling;
      if (!next) {
        return;
      }
      event.preventDefault();
      list.insertBefore(next, li);
    } else {
      return;
    }

    handle.focus();
    persist(li);
  });
})();
