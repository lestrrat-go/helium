<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE17 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of boolean not with @* to test for lack of attributes. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="a[not(@*)]">
       <xsl:value-of select="."/>
	</xsl:for-each>
    <!-- Should output 2, child2, 3. -->
  </out>
</xsl:template>

</xsl:stylesheet>
