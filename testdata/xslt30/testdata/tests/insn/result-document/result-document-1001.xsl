<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for generating two or more result trees with same URI 
  				where the first result tree is implicit and the second tree has an 
  				explicit xsl:result-document with @href="" -->

   <t:output method="xhtml"/>

   <t:template match="/">
      <html>
         <head/>
         <body>First HTML doc in implict xsl:result-document</body>
      </html>
      <t:result-document href="">
         <html>
            <head/>
            <body>Second HTML doc in explicit xsl:result-document</body>
         </html>
      </t:result-document>

   </t:template>
</t:transform>
