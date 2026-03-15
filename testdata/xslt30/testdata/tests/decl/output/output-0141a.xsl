<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="3.0">
<!-- Purpose: Test XHTML output with @escape-uri-attributes="false". Use fn:escape-html-uri to escape some URIs but not all. -->

   <t:output method="xhtml"
             encoding="UTF-8"
             escape-uri-attributes="false"
             indent="no"/>

   <t:template match="/">
      <html>
         <body>
            <t:text>
</t:text>
            <div>
               This is 1:<a href="http://iri.example.org/ﭏ/årsrapport/år/2005?x=y">not escaped</a>
            </div>
            <t:text>
</t:text>
            <div>
This is 2:<a>
                  <t:attribute name="href">
                     <t:value-of select="escape-html-uri('http://iri.example.org/ﭏ/årsrapport/år/2005?x=y')"/>
                  </t:attribute>
               </a>escaped</div>
         </body>
      </html>
   </t:template>
</t:transform>
