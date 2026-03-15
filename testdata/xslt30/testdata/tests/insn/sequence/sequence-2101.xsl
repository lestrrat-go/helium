<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: range expression with functions as operands -->

   <xsl:output method="xml" encoding="UTF-8"/>

   <xsl:template match="/">
      <out>
         <xsl:value-of select="subsequence(doc/num, 3, doc/one) to 10"/>
         <xsl:value-of select="32 to remove(doc/num2, doc/one)"/>
         <xsl:value-of select="remove(doc/num2, 2) to remove(doc/num2, doc/one)"/>
      </out>
   </xsl:template>
</xsl:transform>
