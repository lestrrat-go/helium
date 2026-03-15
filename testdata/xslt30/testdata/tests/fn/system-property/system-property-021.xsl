<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that system-property can be called in static expression.-->

   <xslt:template match="/">
      <out xslt:use-when="system-property('xslt:version') castable as xs:decimal">
         <ok/>        
      </out>
      <out xslt:use-when="not(system-property('xslt:version') castable as xs:decimal)">
         <fail/>        
      </out>
   </xslt:template>
</xslt:transform>
