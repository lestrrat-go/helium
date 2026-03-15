<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- FileName: position91 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.3 -->
    <!-- Purpose: Apply a predicate to a node-set full of attribute nodes.
     If there were no parentheses, we would be asking for the 7th attribute
     named "title" on each element, but there can only be one attribute of a
     particular name on each, so we'd get the empty set. -->

<xsl:output method="xml" encoding="utf-8"/>

<xsl:template match="chapter">
  <out>
    <xsl:text>
</xsl:text>
    <xsl:for-each select="(section//@title)[7]">
      <noted>
        <xsl:value-of select="name(..)"/><xsl:text> </xsl:text>
        <xsl:value-of select="."/>
      </noted>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>


