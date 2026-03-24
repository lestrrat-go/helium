<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output name="temp" method="xhtml" indent="no" encoding="UTF-8"/>
   <!-- Purpose: Test with NCName for @name and xsl:output method=xhtml. 
  				Use the named output definition in xsl:result document.-->

   <t:template match="/">
      <t:result-document format="temp">
 	       <html>
            <body>
               <br/>
               <Option selected="selected"/>
            </body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
