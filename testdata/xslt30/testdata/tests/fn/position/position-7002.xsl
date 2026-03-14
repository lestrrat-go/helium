<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0">

  <!-- FileName: position83 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.2 -->
  <!-- Purpose: Try positional filter on unionpaths that aren't sequential. -->
  
<xsl:template match="/">
  <out>
    <xsl:for-each select="//title">
      Title= <xsl:value-of select="."/>
      <name>
        <xsl:attribute name="first-of">
          <xsl:value-of select="count(ancestor::section
            |../ednote|following::title|../bogus)"/>
        </xsl:attribute>
        <xsl:value-of select="name((ancestor::section|../ednote
          |following::title|../bogus)[1])"/>
      </name>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
