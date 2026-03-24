<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position25 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() function in predicate of wildcard. Look for middle item. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="*[position()=3]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
