<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position96 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Use just a (number-valued) global variable in a predicate. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:variable name="third" select="3"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="a[$third]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
