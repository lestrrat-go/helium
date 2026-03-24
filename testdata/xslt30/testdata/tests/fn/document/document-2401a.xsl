<?xml version="1.0" encoding="UTF-8"?>
<t:package xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:t="http://www.w3.org/1999/XSL/Transform"
                name="http://www.w3.org/xslts/document-2401a"
                package-version="3.0"
                exclude-result-prefixes="xs"
                version="3.0">
<!-- Purpose: Test document() function within a package, overridden by another package
      with different xsl:strip-space declarations. -->
   
   <!-- Note that the results of the document() function with multiple strings as arguments is in document
        order, which is unpredictable -->
   
   
   <t:strip-space elements=""/>
   
   <t:template name="a" visibility="public">
      <unstripped>
         <t:value-of select="count(document('doc14.xml')/*/text())"/>
      </unstripped>
   </t:template>
   
  
   
</t:package>
