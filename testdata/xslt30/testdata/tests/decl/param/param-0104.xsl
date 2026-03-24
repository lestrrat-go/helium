<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test case that uses xsl:sequence instruction to set the value of a variable. -->

   <xslt:variable name="values" as="xs:double*">
      <xslt:sequence select="(1,3,5)"/>
      <xslt:sequence select="(2,4,10)"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:value-of select="sum($values)"/>
      </out>
   </xslt:template>
</xslt:transform>
