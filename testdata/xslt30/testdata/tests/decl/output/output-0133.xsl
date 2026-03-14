<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://myoutputdefinition.example.org"
             xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">

   <t:output name="my:temp" method="xhtml" indent="no" encoding="UTF-8"/>
   <!-- Purpose: Test with prefixed local name for @name and xsl:output 
  				method=xhtml.Use the named output definition in xsl:result document.-->

   <t:template match="/">
      <t:result-document format="my:temp">
 	       <html>
            <body>
               <br/>
               <Option selected="selected"/>
            </body>
         </html>
      </t:result-document>
   </t:template>
</t:transform>
