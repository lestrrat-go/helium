<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- FileName: PREDICATE08 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of implied conversion to number for an inequality test
       in backwards compatibility mode. -->

<xsl:template match="doc">
  <out>
  <xsl:for-each select="a['3.5' &lt; 4]">
    <!-- Always true; strings convert to the numbers they look like. -->
    <xsl:value-of select="."/>
  </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
