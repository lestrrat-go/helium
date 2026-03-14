<?xml version="1.0"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet version="2.0"
     xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <!-- FileName: predicate48 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
  <!-- Purpose: Test (a and b) combination of logical expressions in a predicate. -->
  <!-- Author: Morten Jorgensen -->

<xsl:template match="foo">
<out>
  <test-info>(a and b)</test-info>
  <test-output>
  <xsl:for-each select="bar[@a='1' and @b='1']">
    <xsl:value-of select="@seq"/><xsl:text> </xsl:text>
  </xsl:for-each>
  </test-output>
</out>
</xsl:template>

</xsl:stylesheet>
