<?xml version="1.0" encoding="UTF-8"?>
<t:package xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:t="http://www.w3.org/1999/XSL/Transform"
                name="http://www.w3.org/xslts/override-t-010a"
                package-version="3.0"
                exclude-result-prefixes="xs"
                version="3.0">
<!-- Purpose: It should not be possible to override a template with no declared visibility (default is private) -->
   

   
   <t:template name="a">
      <a>
         <t:value-of select="current-dateTime()"/>
      </a>
   </t:template>
   
  
   
</t:package>
