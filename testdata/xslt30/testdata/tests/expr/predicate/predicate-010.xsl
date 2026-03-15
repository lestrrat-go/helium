<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE29 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
    <!-- Purpose: Test of implied conversion of node-set to number with an inequality relation. -->

<xsl:template match="doc">
  <out>
  <xsl:for-each select = "a[following-sibling::* &lt; 3]">
    <xsl:value-of select="."/>
    <!-- True only when one of the following siblings contains a number less than 3. -->
  </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
