<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0">

  <!-- FileName: position80 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.2 -->
  <!-- Purpose: Ensure last() is working with unionpaths of ancestors. -->
  
<xsl:template match="/">
  <out>
    <xsl:for-each select="//section/title">
      Title= <xsl:value-of select="."/>
      <xsl:variable name="size">
        <xsl:value-of select="count(ancestor::section
          |ancestor::simplesect|ancestor::articleinfo)"/>
      </xsl:variable>
      <xsl:for-each select="(ancestor::section|ancestor::simplesect
        |ancestor::articleinfo)[last()]">
        <name>
          <xsl:attribute name="last-of"><xsl:value-of select="$size"/></xsl:attribute>
          <xsl:value-of select="name(.)"/>
        </name>
      </xsl:for-each>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
