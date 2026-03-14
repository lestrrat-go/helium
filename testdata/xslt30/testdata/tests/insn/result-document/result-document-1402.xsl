<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <t:param name="m" select="'m'" />

   <!-- Purpose: Test JSON serialization, maps with nested HTML -->

   <t:template name="t:initial-template">
      <t:result-document method="json" indent="no" json-node-output-method="ht{$m}l">
         <t:map>
            <t:map-entry key="'doc1'">
               <html>
                  <head>
                     <title>Document 1</title>
                  </head>
                  <body>
                     <p>Content 1</p>
                  </body>
               </html>
            </t:map-entry>
            <t:map-entry key="'doc2'">
               <html>
                  <head>
                     <title>Document 2</title>
                  </head>
                  <body>
                     <p>Content 2</p>
                  </body>
               </html>
            </t:map-entry>
         </t:map>
      </t:result-document>
   </t:template>
</t:transform>
