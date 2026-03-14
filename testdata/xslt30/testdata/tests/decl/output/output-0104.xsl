<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test avoidance of apos entity for XHTML output -->

   <t:output method="xhtml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
      <html xml:lang="en" lang="en">
         <head>
            <title>Virtual Library</title>
         </head>
         <body>
            <p>Moved to <div style='don&apos;t try this'>example.org</div>.</p>
         </body>
      </html>
   </t:template>
</t:transform>
