<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE15 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of boolean and, or with parens overriding precedence. -->

<xsl:template match="doc">
  <out>
  <xsl:for-each select="a[(@squeesh or @squish) and @squash]">
    <xsl:value-of select="."/>
    <!-- Should match 1, 3. -->
  </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
