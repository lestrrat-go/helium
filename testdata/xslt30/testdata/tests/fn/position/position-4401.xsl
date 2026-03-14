<?xml version="1.0" encoding="UTF-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position105 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
  <!-- Creator: Ilene Seelemann -->
  <!-- Purpose: Test last() in a numeric expression and double predicate. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="doc">  
  <out>
    <xsl:apply-templates select="item[last()-1][1]"/> 
    <xsl:text>, </xsl:text>
    <xsl:apply-templates select="item[last()-1][last()]"/> 
  </out> 
</xsl:template>

<xsl:template match="item">
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>