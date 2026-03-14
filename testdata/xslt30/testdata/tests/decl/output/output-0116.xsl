<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml" encoding="UTF-8" indent="no"/>
   <!-- Purpose: Test empty elements with empty content model in XHTML output and @indent=no. -->

   <t:template match="/">
      <html>
         <area tabindex="2"/>
         <base/>
         <basefont/>
         <br/>
         <col/>
         <frame/>
         <hr width="100"/>
         <img/>
         <input/>
         <isindex/>
         <link/>
         <meta/>
         <param/>
      </html>
   </t:template>
</t:transform>
