<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Simple test for constructing an integer sequence -->

   <xslt:variable name="values" as="xs:integer*">
      <xslt:sequence select="(1,2,3,4)"/>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="$values"/>
      </out>
   </xslt:template>
</xslt:transform>
