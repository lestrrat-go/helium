<?xml version="1.1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:x="http://rijk.com/">

<?spec serialization#xml-version?>
  <!-- PURPOSE: Function and variable names containing XML 1.1 name characters -->

<xsl:output method="xml" version="1.0"/>

<xsl:template match="/">
  <doc><xsl:value-of select="x:&#x132;(5)"/></doc>
</xsl:template>

<xsl:function name="x:&#x132;">
 <xsl:param name="x:&#x133;"/>
 <xsl:sequence select="2 + $x:&#x133;"/>
</xsl:function>
 
</xsl:stylesheet>
