<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: First example from spec. -->

   <xslt:variable name="values" as="xs:integer*">
      <xslt:sequence select="(1,2,3,4)"/>
      <xslt:sequence select="(8,9,10)"/>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="sum($values)"/>
      </out>
   </xslt:template>
</xslt:transform>
